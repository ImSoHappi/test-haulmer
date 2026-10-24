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
	merchant, err := s.repo.GetMerchantByID(ctx, req.MerchantID)
	if err != nil {
		return nil, err
	}
	if !merchant.IsActive() {
		return nil, domain.ErrMerchantInactive
	}

	now := time.Now().UTC()
	amount, currency, cardNum, err := s.validateDomainInputs(req, now)
	if err != nil {
		return nil, err
	}

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

	// Idempotency check
	if earlyResp, err := s.checkExistingIdempotency(ctx, req, incomingHash, currency, now); err != nil || earlyResp != nil {
		return earlyResp, err
	}

	// Atomic initial insertion
	txModel, err := s.createInitialTransaction(ctx, req, amount, currency, incomingHash, cardBIN, cardLast4, now)
	if err != nil {
		return nil, err
	}

	// Acquirer authorization & resolution
	return s.executeAcquirerCall(ctx, txModel, req, currency)
}

func (s *Service) validateDomainInputs(req ProcessRequest, now time.Time) (domain.Amount, string, domain.CardNumber, error) {
	amount, err := domain.NewAmount(req.Amount)
	if err != nil {
		return domain.Amount{}, "", domain.CardNumber{}, err
	}
	currency, err := domain.NewCurrency(req.Currency)
	if err != nil {
		return domain.Amount{}, "", domain.CardNumber{}, err
	}
	if err := amount.ValidateMax(s.maxAmountCLP); err != nil {
		return domain.Amount{}, "", domain.CardNumber{}, err
	}
	cardNum, err := domain.NewCardNumber(req.CardNumber)
	if err != nil {
		return domain.Amount{}, "", domain.CardNumber{}, err
	}
	if _, err := domain.NewCVV(req.CardCVV); err != nil {
		return domain.Amount{}, "", domain.CardNumber{}, err
	}
	expiry, err := domain.NewCardExpiry(req.CardExpiryMonth, req.CardExpiryYear)
	if err != nil {
		return domain.Amount{}, "", domain.CardNumber{}, err
	}
	if expiry.IsExpired(now) {
		return domain.Amount{}, "", domain.CardNumber{}, domain.ErrCardExpired
	}
	return amount, currency, cardNum, nil
}

func (s *Service) checkExistingIdempotency(ctx context.Context, req ProcessRequest, incomingHash string, currency string, now time.Time) (*ProcessResponse, error) {
	existingTx, err := s.repo.GetTransactionByIdempotencyKey(ctx, req.MerchantID, req.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency key: %w", err)
	}
	if existingTx == nil {
		return nil, nil
	}
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
		_ = existingTx.MarkFailed("processing_timeout", "Transaction timed out in processing state", now)
		_, _ = s.repo.UpdateTransactionStatus(ctx, nil, existingTx, domain.StatusProcessing)
	}
	return nil, nil
}

func (s *Service) resolvePaymentIntent(
	ctx context.Context,
	tx any,
	req ProcessRequest,
	amount domain.Amount,
	currency string,
	now time.Time,
) (*domain.PaymentIntent, error) {
	pi, err := s.repo.GetPaymentIntentByOrderRef(ctx, req.MerchantID, req.OrderReference)
	if err != nil {
		return nil, err
	}
	if pi != nil {
		hasApproved, err := s.repo.HasApprovedTransaction(ctx, pi.ID)
		if err != nil {
			return nil, err
		}
		if hasApproved {
			return nil, domain.ErrIntentAlreadyApproved
		}
		return pi, nil
	}

	piID := uuid.New()
	pi, err = domain.NewPaymentIntent(piID, req.MerchantID, req.OrderReference, amount, currency, nil, req.CorrelationID, now)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreatePaymentIntent(ctx, tx, pi); err != nil {
		return nil, err
	}
	return pi, nil
}

func (s *Service) createInitialTransaction(
	ctx context.Context,
	req ProcessRequest,
	amount domain.Amount,
	currency string,
	incomingHash, cardBIN, cardLast4 string,
	now time.Time,
) (*domain.Transaction, error) {
	var txModel *domain.Transaction
	err := s.repo.WithTx(ctx, func(tx any) error {
		pi, err := s.resolvePaymentIntent(ctx, tx, req, amount, currency, now)
		if err != nil {
			return err
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

	// Move to PROCESSING
	nowProc := time.Now().UTC()
	if err := txModel.TransitionTo(domain.StatusProcessing, nowProc); err != nil {
		return nil, err
	}
	if _, err := s.repo.UpdateTransactionStatus(ctx, nil, txModel, domain.StatusPending); err != nil {
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
		OccurredAt:    nowProc,
	}
	_ = s.repo.CreateTransactionEvent(ctx, nil, eventProcessing)

	return txModel, nil
}

func (s *Service) executeAcquirerCall(
	ctx context.Context,
	txModel *domain.Transaction,
	req ProcessRequest,
	currency string,
) (*ProcessResponse, error) {
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

	if acqErr != nil {
		if errors.Is(acqErr, domain.ErrAcquirerUnknownResult) {
			return mapToProcessResponse(txModel, req.OrderReference, req.Amount, currency, false), domain.ErrAcquirerUnknownResult
		}
		_ = txModel.MarkFailed("acquirer_failure", acqErr.Error(), nowCompleted)
		_, _ = s.repo.UpdateTransactionStatus(detachedCtx, nil, txModel, domain.StatusProcessing)
		return mapToProcessResponse(txModel, req.OrderReference, req.Amount, currency, false), acqErr
	}

	switch acqResp.Status {
	case "APPROVED":
		_ = txModel.MarkApproved(acqResp.AuthorizationCode, acqResp.Code, "Approved by acquirer", nowCompleted)
	case "DECLINED":
		_ = txModel.MarkDeclined(domain.DeclineSourceAcquirer, acqResp.Code, "Declined by acquirer", nowCompleted)
	default:
		_ = txModel.MarkFailed(acqResp.Code, "Failed by acquirer", nowCompleted)
	}
	if _, err := s.repo.UpdateTransactionStatus(detachedCtx, nil, txModel, domain.StatusProcessing); err != nil {
		_ = txModel.MarkFailed("update_failed", err.Error(), nowCompleted)
		_, _ = s.repo.UpdateTransactionStatus(detachedCtx, nil, txModel, domain.StatusProcessing)
		return nil, err
	}

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
			Payload:   buildEventPayload(e),
			CreatedAt: e.OccurredAt,
		}
	}

	var (
		orderRef string
		amount   int64
		currency string
	)
	if pi, err := s.repo.GetPaymentIntentByID(ctx, tx.PaymentIntentID); err == nil && pi != nil {
		orderRef = pi.OrderReference
		amount = pi.Amount.Int64()
		currency = pi.Currency
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
		OrderReference:    orderRef,
		Amount:            amount,
		Currency:          currency,
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

func buildEventPayload(e *domain.TransactionEvent) map[string]any {
	if e == nil {
		return map[string]any{}
	}
	payload := make(map[string]any, len(e.Details)+7)
	for k, v := range e.Details {
		payload[k] = v
	}
	if e.FromStatus != nil {
		payload["from_status"] = string(*e.FromStatus)
	}
	if e.ToStatus != nil {
		payload["to_status"] = string(*e.ToStatus)
	}
	if e.Code != "" {
		payload["code"] = e.Code
	}
	if e.Message != "" {
		payload["message"] = e.Message
	}
	if e.LatencyMs != nil {
		payload["latency_ms"] = *e.LatencyMs
	}
	if e.AcquirerAttempt != nil {
		payload["acquirer_attempt"] = *e.AcquirerAttempt
	}
	if e.CorrelationID != "" {
		payload["correlation_id"] = e.CorrelationID
	}
	return payload
}
