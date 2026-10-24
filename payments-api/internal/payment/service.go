package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/acquirer"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

const (
	// DefaultLimit is applied when requested limit is <= 0.
	DefaultLimit = 20
	// MaxLimit caps pagination to prevent heavy memory allocations.
	MaxLimit = 50
)

// Service coordinates all payment lifecycle operations: processing, retrieving, and listing.
type Service struct {
	repo         Repository
	acquirer     AcquirerClient
	maxAmountCLP int64
}

// NewService constructs a unified payment service.
func NewService(repo Repository, acquirer AcquirerClient, maxAmountCLP int64) *Service {
	return &Service{
		repo:         repo,
		acquirer:     acquirer,
		maxAmountCLP: maxAmountCLP,
	}
}

// ProcessPayment orchestrates validation, idempotency, persistence, and banking authorization.
func (s *Service) ProcessPayment(ctx context.Context, req ProcessRequest) (*ProcessResponse, error) {
	// 1. Merchant Validation (Early gate / Anti-Carding / Zero DB waste)
	merchant, err := s.repo.GetMerchantByID(ctx, req.MerchantID)
	if err != nil {
		return nil, err
	}
	if !merchant.IsActive() {
		return nil, domain.ErrMerchantInactive
	}

	// 2. Domain Value Objects Validation
	amount, err := domain.NewAmount(req.Amount)
	if err != nil {
		return nil, err
	}
	currency, err := domain.NewCurrency(req.Currency)
	if err != nil {
		return nil, err
	}
	if err := amount.ValidateMax(s.maxAmountCLP); err != nil {
		return nil, err
	}
	cardNum, err := domain.NewCardNumber(req.CardNumber)
	if err != nil {
		return nil, err
	}
	_, err = domain.NewCVV(req.CardCVV)
	if err != nil {
		return nil, err
	}
	expiry, err := domain.NewCardExpiry(req.CardExpiryMonth, req.CardExpiryYear)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if expiry.IsExpired(now) {
		return nil, domain.ErrCardExpired
	}

	// 3. Compute Canonical Payload Hash (PCI compliant, no sensitive PAN or CVV)
	cardBIN := cardNum.BIN()
	cardLast4 := cardNum.Last4()
	incomingHash := domain.ComputePayloadHash(
		req.MerchantID,
		req.OrderReference,
		req.Amount,
		currency,
		cardBIN,
		cardLast4,
		req.CardExpiryMonth,
		req.CardExpiryYear,
	)

	// 4. Idempotency & Concurrency Check
	existingTx, err := s.repo.GetTransactionByIdempotencyKey(ctx, req.MerchantID, req.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency key: %w", err)
	}
	if existingTx != nil {
		if existingTx.PayloadHash != incomingHash {
			return nil, domain.ErrIdempotencyConflict
		}
		if existingTx.Status.IsTerminal() {
			return mapToProcessResponse(existingTx, req.OrderReference, req.Amount, currency, true), nil
		}
		if existingTx.Status == domain.StatusProcessing {
			if time.Since(existingTx.CreatedAt) < 5*time.Minute {
				return nil, domain.ErrActiveTransactionExists
			}
			// Stale/orphan transaction (> 5 min): mark failed to unblock new attempt
			_ = existingTx.MarkFailed("processing_timeout", "Transaction timed out in processing state", now)
			_, _ = s.repo.UpdateTransactionStatus(ctx, nil, existingTx, domain.StatusProcessing)
		}
	}

	// 5. Atomic DB Insertion (PaymentIntent + Transaction PENDING + Audit Event)
	var (
		pi      *domain.PaymentIntent
		txModel *domain.Transaction
	)

	err = s.repo.WithTx(ctx, func(tx any) error {
		var err error
		pi, err = s.repo.GetPaymentIntentByOrderRef(ctx, req.MerchantID, req.OrderReference)
		if err != nil {
			return err
		}
		if pi == nil {
			piID := uuid.New()
			pi, err = domain.NewPaymentIntent(piID, req.MerchantID, req.OrderReference, amount, currency, nil, req.CorrelationID, now)
			if err != nil {
				return err
			}
			if err := s.repo.CreatePaymentIntent(ctx, tx, pi); err != nil {
				return err
			}
		}

		txID := uuid.New()
		txModel = domain.NewTransaction(txID, pi.ID, req.MerchantID, req.IdempotencyKey, incomingHash, cardBIN, cardLast4, req.CorrelationID, now)
		if err := s.repo.CreateTransaction(ctx, tx, txModel); err != nil {
			return err
		}

		eventCreated := &domain.TransactionEvent{
			TransactionID: txModel.ID,
			EventType:     domain.EventTransactionCreated,
			CorrelationID: req.CorrelationID,
			OccurredAt:    now,
			Details: map[string]any{
				"order_reference": req.OrderReference,
				"amount":          req.Amount,
				"currency":        currency,
			},
		}
		return s.repo.CreateTransactionEvent(ctx, tx, eventCreated)
	})
	if err != nil {
		return nil, err
	}

	// 6. Transition to PROCESSING
	nowProcessing := time.Now().UTC()
	if err := txModel.TransitionTo(domain.StatusProcessing, nowProcessing); err != nil {
		return nil, err
	}
	_, err = s.repo.UpdateTransactionStatus(ctx, nil, txModel, domain.StatusPending)
	if err != nil {
		return nil, err
	}

	fromPending := domain.StatusPending
	toProcessing := domain.StatusProcessing
	eventProcessing := &domain.TransactionEvent{
		TransactionID: txModel.ID,
		EventType:     domain.EventStatusChanged,
		FromStatus:    &fromPending,
		ToStatus:      &toProcessing,
		CorrelationID: req.CorrelationID,
		OccurredAt:    nowProcessing,
	}
	_ = s.repo.CreateTransactionEvent(ctx, nil, eventProcessing)

	// 7. Secure Acquirer Authorization with context.WithoutCancel (Anti-Client-Disconnect protection)
	detachedCtx := context.WithoutCancel(ctx)
	startAcq := time.Now()

	authReq := acquirer.AuthorizeRequest{
		ReferenceID:   txModel.ID.String(),
		Amount:        req.Amount,
		Currency:      currency,
		CorrelationID: req.CorrelationID,
		Card: acquirer.CardRequest{
			Number:      req.CardNumber,
			ExpiryMonth: req.CardExpiryMonth,
			ExpiryYear:  req.CardExpiryYear,
			CVV:         req.CardCVV,
			HolderName:  req.CardHolderName,
		},
	}

	acqResp, acqErr := s.acquirer.Authorize(detachedCtx, authReq)
	latencyMs := int(time.Since(startAcq).Milliseconds())
	nowCompleted := time.Now().UTC()

	// 8. Record Acquirer Attempt Event
	attemptEvent := &domain.TransactionEvent{
		TransactionID: txModel.ID,
		EventType:     domain.EventAcquirerAttempt,
		LatencyMs:     &latencyMs,
		CorrelationID: req.CorrelationID,
		OccurredAt:    nowCompleted,
	}
	if acqResp != nil {
		attemptEvent.Code = acqResp.Code
		attemptEvent.Message = acqResp.Status
	}
	_ = s.repo.CreateTransactionEvent(detachedCtx, nil, attemptEvent)

	// 9. Status Resolution & Final Transition
	if acqErr != nil {
		if errors.Is(acqErr, domain.ErrAcquirerUnknownResult) {
			// Leave in PROCESSING, return 202 outcome
			return mapToProcessResponse(txModel, req.OrderReference, req.Amount, currency, false), domain.ErrAcquirerUnknownResult
		}
		_ = txModel.MarkFailed("acquirer_failure", acqErr.Error(), nowCompleted)
		_, _ = s.repo.UpdateTransactionStatus(detachedCtx, nil, txModel, domain.StatusProcessing)
		return mapToProcessResponse(txModel, req.OrderReference, req.Amount, currency, false), acqErr
	}

	if acqResp.Status == "APPROVED" {
		_ = txModel.MarkApproved(acqResp.AuthorizationCode, acqResp.Code, "Approved by acquirer", nowCompleted)
	} else if acqResp.Status == "DECLINED" {
		_ = txModel.MarkDeclined(domain.DeclineSourceAcquirer, acqResp.Code, "Declined by acquirer", nowCompleted)
	} else {
		_ = txModel.MarkFailed(acqResp.Code, "Failed by acquirer", nowCompleted)
	}
	_, _ = s.repo.UpdateTransactionStatus(detachedCtx, nil, txModel, domain.StatusProcessing)

	fromProc := domain.StatusProcessing
	toFinal := txModel.Status
	eventFinal := &domain.TransactionEvent{
		TransactionID: txModel.ID,
		EventType:     domain.EventStatusChanged,
		FromStatus:    &fromProc,
		ToStatus:      &toFinal,
		Code:          txModel.ResultCode,
		Message:       txModel.ResultMessage,
		CorrelationID: req.CorrelationID,
		OccurredAt:    nowCompleted,
	}
	_ = s.repo.CreateTransactionEvent(detachedCtx, nil, eventFinal)

	return mapToProcessResponse(txModel, req.OrderReference, req.Amount, currency, false), nil
}

// GetPayment retrieves transaction details and audit event timeline with tenant isolation.
func (s *Service) GetPayment(ctx context.Context, req GetRequest) (*GetResponse, error) {
	txUUID, err := uuid.Parse(req.TransactionID)
	if err != nil {
		return nil, domain.ErrTransactionNotFound
	}

	tx, err := s.repo.GetTransactionByID(ctx, txUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if tx == nil || tx.MerchantID != req.MerchantID {
		// Strict multi-tenant protection: do not leak transaction existence across merchants
		return nil, domain.ErrTransactionNotFound
	}

	events, err := s.repo.ListEventsByTransactionID(ctx, txUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transaction events: %w", err)
	}

	eventItems := make([]EventItem, len(events))
	for i, e := range events {
		eventItems[i] = EventItem{
			ID:        fmt.Sprintf("%d", e.ID),
			EventType: string(e.EventType),
			Payload:   e.Details,
			CreatedAt: e.OccurredAt,
		}
	}

	return &GetResponse{
		ID:                tx.ID.String(),
		PaymentIntentID:   tx.PaymentIntentID.String(),
		MerchantID:        tx.MerchantID,
		IdempotencyKey:    tx.IdempotencyKey,
		Status:            string(tx.Status),
		CardBIN:           tx.CardBIN,
		CardLast4:         tx.CardLast4,
		DeclineSource:     string(tx.DeclineSource),
		ResultCode:        tx.ResultCode,
		ResultMessage:     tx.ResultMessage,
		AuthorizationCode: tx.AuthorizationCode,
		CorrelationID:     tx.CorrelationID,
		CreatedAt:         tx.CreatedAt,
		UpdatedAt:         tx.UpdatedAt,
		CompletedAt:       tx.CompletedAt,
		Events:            eventItems,
	}, nil
}

// ListPayments retrieves transactions matching the merchant and filter criteria.
func (s *Service) ListPayments(ctx context.Context, req ListRequest) (*ListResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultLimit
	} else if limit > MaxLimit {
		limit = MaxLimit
	}

	offset := max(0, req.Offset)

	merchantID := req.MerchantID
	filter := Filter{
		MerchantID: &merchantID,
		Status:     req.Status,
		Limit:      limit,
		Offset:     offset,
	}

	transactions, totalCount, err := s.repo.ListTransactions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	items := make([]ListItem, len(transactions))
	for i, t := range transactions {
		items[i] = ListItem{
			ID:                t.ID.String(),
			PaymentIntentID:   t.PaymentIntentID.String(),
			MerchantID:        t.MerchantID,
			IdempotencyKey:    t.IdempotencyKey,
			Status:            string(t.Status),
			CardBIN:           t.CardBIN,
			CardLast4:         t.CardLast4,
			DeclineSource:     string(t.DeclineSource),
			ResultCode:        t.ResultCode,
			ResultMessage:     t.ResultMessage,
			AuthorizationCode: t.AuthorizationCode,
			CorrelationID:     t.CorrelationID,
			CreatedAt:         t.CreatedAt,
			UpdatedAt:         t.UpdatedAt,
			CompletedAt:       t.CompletedAt,
		}
	}

	return &ListResponse{
		Items:      items,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

func mapToProcessResponse(t *domain.Transaction, orderRef string, amount int64, currency string, isReplayed bool) *ProcessResponse {
	return &ProcessResponse{
		TransactionID:     t.ID.String(),
		PaymentIntentID:   t.PaymentIntentID.String(),
		MerchantID:        t.MerchantID,
		OrderReference:    orderRef,
		Amount:            amount,
		Currency:          currency,
		Status:            string(t.Status),
		CardBIN:           t.CardBIN,
		CardLast4:         t.CardLast4,
		AuthorizationCode: t.AuthorizationCode,
		DeclineSource:     string(t.DeclineSource),
		ResultCode:        t.ResultCode,
		ResultMessage:     t.ResultMessage,
		IsReplayed:        isReplayed,
		CreatedAt:         t.CreatedAt,
		CompletedAt:       t.CompletedAt,
	}
}
