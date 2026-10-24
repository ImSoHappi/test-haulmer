package getpayment

import (
	"context"

	"github.com/google/uuid"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// Repository defines persistence operations needed by the GetPayment service.
type Repository interface {
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	ListEventsByTransactionID(ctx context.Context, txID uuid.UUID) ([]*domain.TransactionEvent, error)
}
