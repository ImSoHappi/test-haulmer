package payment

import (
	"context"

	"github.com/google/uuid"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/acquirer"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// Repository defines secondary persistence ports required by the payment service.
type Repository interface {
	GetMerchantByID(ctx context.Context, id string) (*domain.Merchant, error)
	GetPaymentIntentByOrderRef(ctx context.Context, merchantID, orderRef string) (*domain.PaymentIntent, error)
	GetTransactionByIdempotencyKey(ctx context.Context, merchantID, key string) (*domain.Transaction, error)
	WithTx(ctx context.Context, fn func(tx any) error) error
	CreatePaymentIntent(ctx context.Context, tx any, pi *domain.PaymentIntent) error
	CreateTransaction(ctx context.Context, tx any, t *domain.Transaction) error
	CreateTransactionEvent(ctx context.Context, tx any, e *domain.TransactionEvent) error
	UpdateTransactionStatus(ctx context.Context, tx any, t *domain.Transaction, fromStatus domain.TransactionStatus) (bool, error)
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	ListEventsByTransactionID(ctx context.Context, txID uuid.UUID) ([]*domain.TransactionEvent, error)
	ListTransactions(ctx context.Context, filter Filter) ([]*domain.Transaction, int, error)
	HasApprovedTransaction(ctx context.Context, intentID uuid.UUID) (bool, error)
}

// AcquirerClient defines the secondary banking client required to authorize charges.
type AcquirerClient interface {
	Authorize(ctx context.Context, req acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error)
}
