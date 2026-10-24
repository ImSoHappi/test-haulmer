package getpayment

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// Service coordinates retrieving payment details and audit history.
type Service struct {
	repo Repository
}

// NewService constructs a new GetPayment service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetPayment retrieves transaction details and audit event timeline with tenant isolation.
func (s *Service) GetPayment(ctx context.Context, req Request) (*Response, error) {
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

	return &Response{
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
