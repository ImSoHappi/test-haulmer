package listpayments

import (
	"context"
	"fmt"
)

const (
	// DefaultLimit is applied when requested limit is <= 0.
	DefaultLimit = 20
	// MaxLimit caps pagination to prevent heavy memory allocations.
	MaxLimit = 50
)

// Service coordinates listing transactions for a merchant with filtering and pagination.
type Service struct {
	repo Repository
}

// NewService constructs a new ListPayments service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListPayments retrieves transactions matching the merchant and filter criteria.
func (s *Service) ListPayments(ctx context.Context, req Request) (*Response, error) {
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

	items := make([]Item, len(transactions))
	for i, t := range transactions {
		items[i] = Item{
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

	return &Response{
		Items:      items,
		TotalCount: totalCount,
		Limit:      limit,
		Offset:     offset,
	}, nil
}
