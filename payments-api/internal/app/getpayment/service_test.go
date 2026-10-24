package getpayment_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/app/getpayment"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

type MockGetPaymentRepository struct {
	GetTransactionByIDFunc        func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	ListEventsByTransactionIDFunc func(ctx context.Context, txID uuid.UUID) ([]*domain.TransactionEvent, error)
}

func (m *MockGetPaymentRepository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	if m.GetTransactionByIDFunc != nil {
		return m.GetTransactionByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockGetPaymentRepository) ListEventsByTransactionID(ctx context.Context, txID uuid.UUID) ([]*domain.TransactionEvent, error) {
	if m.ListEventsByTransactionIDFunc != nil {
		return m.ListEventsByTransactionIDFunc(ctx, txID)
	}
	return nil, nil
}

func TestService_GetPayment_Success(t *testing.T) {
	now := time.Now().UTC()
	txID := uuid.New()
	piID := uuid.New()

	tx := domain.NewTransaction(txID, piID, "mer-1", "idem-1", "hash-1", "424242", "4242", "corr-1", now)
	_ = tx.TransitionTo(domain.StatusProcessing, now)
	_ = tx.MarkApproved("AUT-100", "00", "Approved", now)

	events := []*domain.TransactionEvent{
		{ID: 1, TransactionID: txID, EventType: domain.EventTransactionCreated, OccurredAt: now, Details: map[string]any{"amount": int64(1000)}},
		{ID: 2, TransactionID: txID, EventType: domain.EventStatusChanged, OccurredAt: now},
		{ID: 3, TransactionID: txID, EventType: domain.EventStatusChanged, OccurredAt: now},
	}

	repo := &MockGetPaymentRepository{
		GetTransactionByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
			return tx, nil
		},
		ListEventsByTransactionIDFunc: func(ctx context.Context, txID uuid.UUID) ([]*domain.TransactionEvent, error) {
			return events, nil
		},
	}

	svc := getpayment.NewService(repo)
	out, err := svc.GetPayment(context.Background(), getpayment.Request{
		MerchantID:    "mer-1",
		TransactionID: txID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, txID.String(), out.ID)
	assert.Equal(t, "APPROVED", out.Status)
	assert.Equal(t, "AUT-100", out.AuthorizationCode)
	assert.Len(t, out.Events, 3)
	assert.Equal(t, string(domain.EventTransactionCreated), out.Events[0].EventType)
}

func TestService_GetPayment_NotFound(t *testing.T) {
	repo := &MockGetPaymentRepository{}
	svc := getpayment.NewService(repo)

	_, err := svc.GetPayment(context.Background(), getpayment.Request{
		MerchantID:    "mer-1",
		TransactionID: uuid.New().String(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTransactionNotFound)
}

func TestService_GetPayment_TenantIsolation(t *testing.T) {
	now := time.Now().UTC()
	txID := uuid.New()
	// Transaction belongs to mer-2, but requested by mer-1
	tx := domain.NewTransaction(txID, uuid.New(), "mer-2", "idem-2", "hash-2", "424242", "4242", "corr-2", now)

	repo := &MockGetPaymentRepository{
		GetTransactionByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
			return tx, nil
		},
	}

	svc := getpayment.NewService(repo)
	_, err := svc.GetPayment(context.Background(), getpayment.Request{
		MerchantID:    "mer-1", // Attacker / wrong merchant
		TransactionID: txID.String(),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTransactionNotFound, "Must not reveal other merchant transactions")
}
