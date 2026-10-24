package listpayments

import (
	"context"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// Repository defines persistence operations needed by the ListPayments service.
type Repository interface {
	ListTransactions(ctx context.Context, filter Filter) ([]*domain.Transaction, int, error)
}
