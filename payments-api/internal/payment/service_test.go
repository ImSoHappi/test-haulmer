package payment_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/acquirer"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/payment"
)

// MockRepository implements payment.Repository for unit tests
type MockRepository struct {
	GetMerchantByIDFunc                func(ctx context.Context, id string) (*domain.Merchant, error)
	GetPaymentIntentByOrderRefFunc     func(ctx context.Context, merchantID, orderRef string) (*domain.PaymentIntent, error)
	GetTransactionByIdempotencyKeyFunc func(ctx context.Context, merchantID, key string) (*domain.Transaction, error)
	WithTxFunc                         func(ctx context.Context, fn func(tx any) error) error
	CreatePaymentIntentFunc            func(ctx context.Context, tx any, pi *domain.PaymentIntent) error
	CreateTransactionFunc              func(ctx context.Context, tx any, t *domain.Transaction) error
	CreateTransactionEventFunc         func(ctx context.Context, tx any, e *domain.TransactionEvent) error
	UpdateTransactionStatusFunc        func(ctx context.Context, tx any, t *domain.Transaction, fromStatus domain.TransactionStatus) (bool, error)
	GetTransactionByIDFunc             func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	ListEventsByTransactionIDFunc      func(ctx context.Context, txID uuid.UUID) ([]*domain.TransactionEvent, error)
	ListTransactionsFunc               func(ctx context.Context, filter payment.Filter) ([]*domain.Transaction, int, error)
	HasApprovedTransactionFunc         func(ctx context.Context, intentID uuid.UUID) (bool, error)
	GetPaymentIntentByIDFunc           func(ctx context.Context, id uuid.UUID) (*domain.PaymentIntent, error)
}

func (m *MockRepository) GetPaymentIntentByID(ctx context.Context, id uuid.UUID) (*domain.PaymentIntent, error) {
	if m.GetPaymentIntentByIDFunc != nil {
		return m.GetPaymentIntentByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockRepository) GetMerchantByID(ctx context.Context, id string) (*domain.Merchant, error) {
	if m.GetMerchantByIDFunc != nil {
		return m.GetMerchantByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockRepository) GetPaymentIntentByOrderRef(ctx context.Context, merchantID, orderRef string) (*domain.PaymentIntent, error) {
	if m.GetPaymentIntentByOrderRefFunc != nil {
		return m.GetPaymentIntentByOrderRefFunc(ctx, merchantID, orderRef)
	}
	return nil, nil
}

func (m *MockRepository) GetTransactionByIdempotencyKey(ctx context.Context, merchantID, key string) (*domain.Transaction, error) {
	if m.GetTransactionByIdempotencyKeyFunc != nil {
		return m.GetTransactionByIdempotencyKeyFunc(ctx, merchantID, key)
	}
	return nil, nil
}

func (m *MockRepository) WithTx(ctx context.Context, fn func(tx any) error) error {
	if m.WithTxFunc != nil {
		return m.WithTxFunc(ctx, fn)
	}
	return fn(nil)
}

func (m *MockRepository) CreatePaymentIntent(ctx context.Context, tx any, pi *domain.PaymentIntent) error {
	if m.CreatePaymentIntentFunc != nil {
		return m.CreatePaymentIntentFunc(ctx, tx, pi)
	}
	return nil
}

func (m *MockRepository) CreateTransaction(ctx context.Context, tx any, t *domain.Transaction) error {
	if m.CreateTransactionFunc != nil {
		return m.CreateTransactionFunc(ctx, tx, t)
	}
	return nil
}

func (m *MockRepository) CreateTransactionEvent(ctx context.Context, tx any, e *domain.TransactionEvent) error {
	if m.CreateTransactionEventFunc != nil {
		return m.CreateTransactionEventFunc(ctx, tx, e)
	}
	return nil
}

func (m *MockRepository) UpdateTransactionStatus(ctx context.Context, tx any, t *domain.Transaction, fromStatus domain.TransactionStatus) (bool, error) {
	if m.UpdateTransactionStatusFunc != nil {
		return m.UpdateTransactionStatusFunc(ctx, tx, t, fromStatus)
	}
	return true, nil
}

func (m *MockRepository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	if m.GetTransactionByIDFunc != nil {
		return m.GetTransactionByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockRepository) ListEventsByTransactionID(ctx context.Context, txID uuid.UUID) ([]*domain.TransactionEvent, error) {
	if m.ListEventsByTransactionIDFunc != nil {
		return m.ListEventsByTransactionIDFunc(ctx, txID)
	}
	return nil, nil
}

func (m *MockRepository) ListTransactions(ctx context.Context, filter payment.Filter) ([]*domain.Transaction, int, error) {
	if m.ListTransactionsFunc != nil {
		return m.ListTransactionsFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *MockRepository) HasApprovedTransaction(ctx context.Context, intentID uuid.UUID) (bool, error) {
	if m.HasApprovedTransactionFunc != nil {
		return m.HasApprovedTransactionFunc(ctx, intentID)
	}
	return false, nil
}

// MockAcquirerClient implements payment.AcquirerClient for unit tests
type MockAcquirerClient struct {
	AuthorizeFunc func(ctx context.Context, req acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error)
}

func (m *MockAcquirerClient) Authorize(ctx context.Context, req acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error) {
	if m.AuthorizeFunc != nil {
		return m.AuthorizeFunc(ctx, req)
	}
	return nil, nil
}

func validSampleRequest() payment.ProcessRequest {
	return payment.ProcessRequest{
		MerchantID:      "1001",
		IdempotencyKey:  "idem-valid-key-12345",
		OrderReference:  "KD0877",
		Amount:          25000,
		Currency:        "CLP",
		CardNumber:      "4242424242424242",
		CardExpiryMonth: 12,
		CardExpiryYear:  2030,
		CardCVV:         "123",
		CardHolderName:  "Ada Lovelace",
		CorrelationID:   "corr-12345",
	}
}

func TestService_ProcessPayment_Success(t *testing.T) {
	repo := &MockRepository{
		GetMerchantByIDFunc: func(ctx context.Context, id string) (*domain.Merchant, error) {
			m := domain.NewMerchant(id, "Test Merchant", domain.MerchantStatusActive)
			return &m, nil
		},
	}
	acq := &MockAcquirerClient{
		AuthorizeFunc: func(ctx context.Context, req acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error) {
			return &acquirer.AcquirerResponse{
				Status:            "APPROVED",
				Code:              "00",
				AuthorizationCode: "123456",
			}, nil
		},
	}

	svc := payment.NewService(repo, acq, 10000000)
	resp, err := svc.ProcessPayment(context.Background(), validSampleRequest())

	require.NoError(t, err)
	assert.Equal(t, "APPROVED", resp.Status)
	assert.Equal(t, "123456", resp.AuthorizationCode)
	assert.False(t, resp.IsReplayed)
}

func TestService_ProcessPayment_IdempotencyReplay(t *testing.T) {
	now := time.Now().UTC()
	req := validSampleRequest()

	currency, _ := domain.NewCurrency(req.Currency)
	cardNum, _ := domain.NewCardNumber(req.CardNumber)
	incomingHash := domain.ComputePayloadHash(
		req.MerchantID,
		req.OrderReference,
		req.Amount,
		currency,
		cardNum.BIN(),
		cardNum.Last4(),
		req.CardExpiryMonth,
		req.CardExpiryYear,
	)

	existingTx := domain.NewTransaction(uuid.New(), uuid.New(), req.MerchantID, req.IdempotencyKey, incomingHash, cardNum.BIN(), cardNum.Last4(), req.CorrelationID, now)
	_ = existingTx.TransitionTo(domain.StatusProcessing, now)
	_ = existingTx.MarkApproved("AUTH-EXISTING", "00", "Approved", now)

	repo := &MockRepository{
		GetMerchantByIDFunc: func(ctx context.Context, id string) (*domain.Merchant, error) {
			m := domain.NewMerchant(id, "Test Merchant", domain.MerchantStatusActive)
			return &m, nil
		},
		GetTransactionByIdempotencyKeyFunc: func(ctx context.Context, merchantID, key string) (*domain.Transaction, error) {
			return existingTx, nil
		},
	}

	svc := payment.NewService(repo, &MockAcquirerClient{}, 10000000)
	resp, err := svc.ProcessPayment(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.IsReplayed)
	assert.Equal(t, "APPROVED", resp.Status)
	assert.Equal(t, "AUTH-EXISTING", resp.AuthorizationCode)
}

func TestService_ProcessPayment_MerchantInactive(t *testing.T) {
	repo := &MockRepository{
		GetMerchantByIDFunc: func(ctx context.Context, id string) (*domain.Merchant, error) {
			m := domain.NewMerchant(id, "Inactive Merchant", domain.MerchantStatusInactive)
			return &m, nil
		},
	}

	svc := payment.NewService(repo, &MockAcquirerClient{}, 10000000)
	_, err := svc.ProcessPayment(context.Background(), validSampleRequest())

	assert.ErrorIs(t, err, domain.ErrMerchantInactive)
}

func TestService_ProcessPayment_AmountExceedsLimit(t *testing.T) {
	repo := &MockRepository{
		GetMerchantByIDFunc: func(ctx context.Context, id string) (*domain.Merchant, error) {
			m := domain.NewMerchant(id, "Test Merchant", domain.MerchantStatusActive)
			return &m, nil
		},
	}

	svc := payment.NewService(repo, &MockAcquirerClient{}, 5000) // Limit is 5.000, input is 25.000
	_, err := svc.ProcessPayment(context.Background(), validSampleRequest())

	assert.ErrorIs(t, err, domain.ErrAmountExceedsLimit)
}

func TestService_ProcessPayment_CardExpired(t *testing.T) {
	repo := &MockRepository{
		GetMerchantByIDFunc: func(ctx context.Context, id string) (*domain.Merchant, error) {
			m := domain.NewMerchant(id, "Test Merchant", domain.MerchantStatusActive)
			return &m, nil
		},
	}

	req := validSampleRequest()
	req.CardExpiryMonth = 1
	req.CardExpiryYear = 2020

	svc := payment.NewService(repo, &MockAcquirerClient{}, 10000000)
	_, err := svc.ProcessPayment(context.Background(), req)

	assert.ErrorIs(t, err, domain.ErrCardExpired)
}

func TestService_ProcessPayment_DeclinedByAcquirer(t *testing.T) {
	repo := &MockRepository{
		GetMerchantByIDFunc: func(ctx context.Context, id string) (*domain.Merchant, error) {
			m := domain.NewMerchant(id, "Test Merchant", domain.MerchantStatusActive)
			return &m, nil
		},
	}
	acq := &MockAcquirerClient{
		AuthorizeFunc: func(ctx context.Context, req acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error) {
			return &acquirer.AcquirerResponse{
				Status: "DECLINED",
				Code:   "51", // Insufficient funds
			}, nil
		},
	}

	svc := payment.NewService(repo, acq, 10000000)
	resp, err := svc.ProcessPayment(context.Background(), validSampleRequest())

	require.NoError(t, err)
	assert.Equal(t, "DECLINED", resp.Status)
	assert.Equal(t, string(domain.DeclineSourceAcquirer), resp.DeclineSource)
	assert.Equal(t, "51", resp.ResultCode)
}

func TestService_GetPayment_Success(t *testing.T) {
	now := time.Now().UTC()
	txID := uuid.New()
	piID := uuid.New()
	merchantID := "1001"

	tx := domain.NewTransaction(txID, piID, merchantID, "idem-1", "hash-1", "424242", "4242", "corr-1", now)
	_ = tx.TransitionTo(domain.StatusProcessing, now)
	_ = tx.MarkApproved("AUTH-999", "00", "Approved", now)

	repo := &MockRepository{
		GetTransactionByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
			if id == txID {
				return tx, nil
			}
			return nil, nil
		},
		GetPaymentIntentByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.PaymentIntent, error) {
			amt, _ := domain.NewAmount(10000)
			return domain.NewPaymentIntent(id, merchantID, "ORD-TEST-1", amt, "CLP", nil, "corr-test", now)
		},
		ListEventsByTransactionIDFunc: func(ctx context.Context, id uuid.UUID) ([]*domain.TransactionEvent, error) {
			fromSt := domain.StatusPending
			toSt := domain.StatusProcessing
			code := "51"
			msg := "Fondos insuficientes"
			latency := 150
			attempt := 1
			return []*domain.TransactionEvent{
				{
					ID:              1,
					TransactionID:   txID,
					EventType:       domain.EventStatusChanged,
					FromStatus:      &fromSt,
					ToStatus:        &toSt,
					Code:            code,
					Message:         msg,
					LatencyMs:       &latency,
					AcquirerAttempt: &attempt,
					CorrelationID:   "corr-event-1",
					OccurredAt:      now,
					Details:         map[string]any{"custom": "value"},
				},
			}, nil
		},
	}

	svc := payment.NewService(repo, &MockAcquirerClient{}, 10000000)
	out, err := svc.GetPayment(context.Background(), payment.GetRequest{
		MerchantID:    merchantID,
		TransactionID: txID.String(),
	})

	require.NoError(t, err)
	assert.Equal(t, txID.String(), out.ID)
	assert.Equal(t, "APPROVED", out.Status)
	assert.Equal(t, "AUTH-999", out.AuthorizationCode)
	assert.Equal(t, "ORD-TEST-1", out.OrderReference)
	assert.Equal(t, int64(10000), out.Amount)
	assert.Equal(t, "CLP", out.Currency)
	assert.Len(t, out.Events, 1)
	assert.Equal(t, string(domain.EventStatusChanged), out.Events[0].EventType)
	assert.Equal(t, "PENDING", out.Events[0].Payload["from_status"])
	assert.Equal(t, "PROCESSING", out.Events[0].Payload["to_status"])
	assert.Equal(t, "51", out.Events[0].Payload["code"])
	assert.Equal(t, "Fondos insuficientes", out.Events[0].Payload["message"])
	assert.Equal(t, 150, out.Events[0].Payload["latency_ms"])
	assert.Equal(t, 1, out.Events[0].Payload["acquirer_attempt"])
	assert.Equal(t, "corr-event-1", out.Events[0].Payload["correlation_id"])
	assert.Equal(t, "value", out.Events[0].Payload["custom"])
}

func TestService_GetPayment_MultiTenantIsolation(t *testing.T) {
	now := time.Now().UTC()
	txID := uuid.New()
	piID := uuid.New()

	tx := domain.NewTransaction(txID, piID, "1001", "idem-1", "hash-1", "424242", "4242", "corr-1", now)

	repo := &MockRepository{
		GetTransactionByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
			return tx, nil
		},
	}

	svc := payment.NewService(repo, &MockAcquirerClient{}, 10000000)
	// Request with a different merchant ID (1002) should not find the transaction of merchant 1001
	_, err := svc.GetPayment(context.Background(), payment.GetRequest{
		MerchantID:    "1002",
		TransactionID: txID.String(),
	})

	assert.ErrorIs(t, err, domain.ErrTransactionNotFound)
}

func TestService_ListPayments_Success(t *testing.T) {
	now := time.Now().UTC()
	txID := uuid.New()
	piID := uuid.New()
	merchantID := "1001"

	tx := domain.NewTransaction(txID, piID, merchantID, "idem-1", "hash-1", "424242", "4242", "corr-1", now)
	_ = tx.TransitionTo(domain.StatusProcessing, now)
	_ = tx.MarkApproved("AUTH-1", "00", "Approved", now)

	var capturedFilter payment.Filter
	repo := &MockRepository{
		ListTransactionsFunc: func(ctx context.Context, filter payment.Filter) ([]*domain.Transaction, int, error) {
			capturedFilter = filter
			return []*domain.Transaction{tx}, 1, nil
		},
	}

	svc := payment.NewService(repo, &MockAcquirerClient{}, 10000000)
	out, err := svc.ListPayments(context.Background(), payment.ListRequest{
		MerchantID: merchantID,
		Limit:      10,
		Offset:     5,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, out.TotalCount)
	assert.Equal(t, 10, out.Limit)
	assert.Equal(t, 5, out.Offset)
	assert.Len(t, out.Items, 1)
	assert.Equal(t, txID.String(), out.Items[0].ID)
	assert.Equal(t, "APPROVED", out.Items[0].Status)
	assert.Equal(t, 10, capturedFilter.Limit)
	assert.Equal(t, 5, capturedFilter.Offset)
	assert.Equal(t, merchantID, *capturedFilter.MerchantID)
}

func TestService_ListPayments_PaginationDefaults(t *testing.T) {
	var capturedFilter payment.Filter
	repo := &MockRepository{
		ListTransactionsFunc: func(ctx context.Context, filter payment.Filter) ([]*domain.Transaction, int, error) {
			capturedFilter = filter
			return nil, 0, nil
		},
	}

	svc := payment.NewService(repo, &MockAcquirerClient{}, 10000000)

	// Case 1: Limit <= 0 defaults to 20, Offset < 0 clamped to 0
	_, err := svc.ListPayments(context.Background(), payment.ListRequest{
		MerchantID: "1001",
		Limit:      0,
		Offset:     -10,
	})
	require.NoError(t, err)
	assert.Equal(t, 20, capturedFilter.Limit)
	assert.Equal(t, 0, capturedFilter.Offset)

	// Case 2: Limit > 50 capped to 50
	_, err = svc.ListPayments(context.Background(), payment.ListRequest{
		MerchantID: "1001",
		Limit:      100,
		Offset:     10,
	})
	require.NoError(t, err)
	assert.Equal(t, 50, capturedFilter.Limit)
	assert.Equal(t, 10, capturedFilter.Offset)
}
