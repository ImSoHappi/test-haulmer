package listpayments_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/app/listpayments"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

type MockListPaymentRepository struct {
	ListTransactionsFunc func(ctx context.Context, filter listpayments.Filter) ([]*domain.Transaction, int, error)
}

func (m *MockListPaymentRepository) ListTransactions(ctx context.Context, filter listpayments.Filter) ([]*domain.Transaction, int, error) {
	if m.ListTransactionsFunc != nil {
		return m.ListTransactionsFunc(ctx, filter)
	}
	return nil, 0, nil
}

func TestService_ListPayments_Success(t *testing.T) {
	now := time.Now().UTC()
	tx1 := domain.NewTransaction(uuid.New(), uuid.New(), "mer-1", "idem-1", "hash-1", "424242", "4242", "corr-1", now)
	_ = tx1.TransitionTo(domain.StatusProcessing, now)
	_ = tx1.MarkApproved("AUT-1", "00", "Approved", now)

	tx2 := domain.NewTransaction(uuid.New(), uuid.New(), "mer-1", "idem-2", "hash-2", "424242", "4242", "corr-2", now)
	_ = tx2.TransitionTo(domain.StatusProcessing, now)
	_ = tx2.MarkDeclined(domain.DeclineSourceAcquirer, "51", "Insufficient funds", now)

	var capturedFilter listpayments.Filter
	repo := &MockListPaymentRepository{
		ListTransactionsFunc: func(ctx context.Context, filter listpayments.Filter) ([]*domain.Transaction, int, error) {
			capturedFilter = filter
			return []*domain.Transaction{tx1, tx2}, 2, nil
		},
	}

	svc := listpayments.NewService(repo)
	out, err := svc.ListPayments(context.Background(), listpayments.Request{
		MerchantID: "mer-1",
		Limit:      10,
		Offset:     0,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, out.TotalCount)
	assert.Len(t, out.Items, 2)
	assert.Equal(t, "mer-1", *capturedFilter.MerchantID)
	assert.Equal(t, 10, capturedFilter.Limit)
	assert.Equal(t, 0, capturedFilter.Offset)
	assert.Equal(t, "APPROVED", out.Items[0].Status)
	assert.Equal(t, "DECLINED", out.Items[1].Status)
}

func TestService_ListPayments_PaginationDefaults(t *testing.T) {
	var capturedFilter listpayments.Filter
	repo := &MockListPaymentRepository{
		ListTransactionsFunc: func(ctx context.Context, filter listpayments.Filter) ([]*domain.Transaction, int, error) {
			capturedFilter = filter
			return []*domain.Transaction{}, 0, nil
		},
	}

	svc := listpayments.NewService(repo)

	// Case 1: Limit <= 0 should default to 20
	_, err := svc.ListPayments(context.Background(), listpayments.Request{
		MerchantID: "mer-1",
		Limit:      0,
		Offset:     -5,
	})
	require.NoError(t, err)
	assert.Equal(t, 20, capturedFilter.Limit)
	assert.Equal(t, 0, capturedFilter.Offset)

	// Case 2: Limit > MaxLimit (50) should cap at 50
	_, err = svc.ListPayments(context.Background(), listpayments.Request{
		MerchantID: "mer-1",
		Limit:      500,
	})
	require.NoError(t, err)
	assert.Equal(t, 50, capturedFilter.Limit)
}

func TestService_ListPayments_FilterByStatus(t *testing.T) {
	var capturedFilter listpayments.Filter
	repo := &MockListPaymentRepository{
		ListTransactionsFunc: func(ctx context.Context, filter listpayments.Filter) ([]*domain.Transaction, int, error) {
			capturedFilter = filter
			return []*domain.Transaction{}, 0, nil
		},
	}

	svc := listpayments.NewService(repo)
	statusApproved := domain.StatusApproved

	_, err := svc.ListPayments(context.Background(), listpayments.Request{
		MerchantID: "mer-1",
		Status:     &statusApproved,
	})
	require.NoError(t, err)
	require.NotNil(t, capturedFilter.Status)
	assert.Equal(t, domain.StatusApproved, *capturedFilter.Status)
}
