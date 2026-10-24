package processpayment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/acquirer"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/app/processpayment"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// MockRepository implements processpayment.Repository for unit tests
type MockRepository struct {
	GetMerchantByIDFunc                 func(ctx context.Context, id string) (*domain.Merchant, error)
	GetPaymentIntentByOrderRefFunc      func(ctx context.Context, merchantID, orderRef string) (*domain.PaymentIntent, error)
	GetTransactionByIdempotencyKeyFunc  func(ctx context.Context, merchantID, key string) (*domain.Transaction, error)
	WithTxFunc                          func(ctx context.Context, fn func(tx any) error) error
	CreatePaymentIntentFunc             func(ctx context.Context, tx any, pi *domain.PaymentIntent) error
	CreateTransactionFunc               func(ctx context.Context, tx any, t *domain.Transaction) error
	CreateTransactionEventFunc          func(ctx context.Context, tx any, e *domain.TransactionEvent) error
	UpdateTransactionStatusFunc         func(ctx context.Context, tx any, t *domain.Transaction, originalStatus domain.TransactionStatus) (bool, error)
}

func (m *MockRepository) GetMerchantByID(ctx context.Context, id string) (*domain.Merchant, error) {
	if m.GetMerchantByIDFunc != nil {
		return m.GetMerchantByIDFunc(ctx, id)
	}
	merchant := domain.NewMerchant(id, "Test Merchant", domain.MerchantStatusActive)
	return &merchant, nil
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

func (m *MockRepository) UpdateTransactionStatus(ctx context.Context, tx any, t *domain.Transaction, originalStatus domain.TransactionStatus) (bool, error) {
	if m.UpdateTransactionStatusFunc != nil {
		return m.UpdateTransactionStatusFunc(ctx, tx, t, originalStatus)
	}
	return true, nil
}

// MockAcquirerClient implements processpayment.AcquirerClient for unit tests
type MockAcquirerClient struct {
	AuthorizeFunc func(ctx context.Context, req acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error)
}

func (m *MockAcquirerClient) Authorize(ctx context.Context, req acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error) {
	if m.AuthorizeFunc != nil {
		return m.AuthorizeFunc(ctx, req)
	}
	return &acquirer.AcquirerResponse{
		ReferenceID:       req.ReferenceID,
		Status:            "APPROVED",
		Code:              "00",
		AuthorizationCode: "AUT-TEST-999",
		Amount:            req.Amount,
		Currency:          req.Currency,
		CreatedAt:         time.Now(),
	}, nil
}

func validSampleRequest() processpayment.Request {
	return processpayment.Request{
		MerchantID:      "mer-test-1",
		IdempotencyKey:  "idem-test-1",
		OrderReference:  "ORD-9876",
		Amount:          25000,
		Currency:        "CLP",
		CardNumber:      "4242424242424242",
		CardExpiryMonth: 12,
		CardExpiryYear:  2028,
		CardCVV:         "123",
		CardHolderName:  "Test Buyer",
		CorrelationID:   "corr-12345",
	}
}

// 1. Success Approved
func TestService_ProcessPayment_Success_Approved(t *testing.T) {
	repo := &MockRepository{}
	acq := &MockAcquirerClient{}

	svc := processpayment.NewService(repo, acq, 10000000)
	out, err := svc.ProcessPayment(context.Background(), validSampleRequest())

	require.NoError(t, err)
	assert.Equal(t, "APPROVED", out.Status)
	assert.Equal(t, "AUT-TEST-999", out.AuthorizationCode)
	assert.False(t, out.IsReplayed)
	assert.Equal(t, "424242", out.CardBIN)
	assert.Equal(t, "4242", out.CardLast4)
}

// 2. Success Declined by Acquirer
func TestService_ProcessPayment_Success_Declined(t *testing.T) {
	repo := &MockRepository{}
	acq := &MockAcquirerClient{
		AuthorizeFunc: func(ctx context.Context, req acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error) {
			return &acquirer.AcquirerResponse{
				ReferenceID: req.ReferenceID,
				Status:      "DECLINED",
				Code:        "51",
			}, nil
		},
	}

	svc := processpayment.NewService(repo, acq, 10000000)
	out, err := svc.ProcessPayment(context.Background(), validSampleRequest())

	require.NoError(t, err)
	assert.Equal(t, "DECLINED", out.Status)
	assert.Equal(t, "51", out.ResultCode)
}

// 3. Merchant Inactive
func TestService_ProcessPayment_MerchantInactive(t *testing.T) {
	repo := &MockRepository{
		GetMerchantByIDFunc: func(ctx context.Context, id string) (*domain.Merchant, error) {
			m := domain.NewMerchant(id, "Inactive Merchant", domain.MerchantStatusInactive)
			return &m, nil
		},
	}

	svc := processpayment.NewService(repo, &MockAcquirerClient{}, 10000000)
	_, err := svc.ProcessPayment(context.Background(), validSampleRequest())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrMerchantInactive)
}

// 4. Amount Limit Exceeded
func TestService_ProcessPayment_AmountLimitExceeded(t *testing.T) {
	repo := &MockRepository{}
	svc := processpayment.NewService(repo, &MockAcquirerClient{}, 5000) // Limit is 5.000, input is 25.000

	_, err := svc.ProcessPayment(context.Background(), validSampleRequest())
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAmountExceedsLimit)
}

// 5. Idempotent Replay (already approved)
func TestService_ProcessPayment_Idempotent_Replay(t *testing.T) {
	req := validSampleRequest()
	now := time.Now().UTC()
	hash := domain.ComputePayloadHash(req.MerchantID, req.OrderReference, req.Amount, req.Currency, "424242", "4242", req.CardExpiryMonth, req.CardExpiryYear)

	existingTx := domain.NewTransaction(uuid.New(), uuid.New(), req.MerchantID, req.IdempotencyKey, hash, "424242", "4242", "corr-prev", now)
	_ = existingTx.TransitionTo(domain.StatusProcessing, now)
	_ = existingTx.MarkApproved("AUT-PREV", "00", "Approved", now)

	repo := &MockRepository{
		GetTransactionByIdempotencyKeyFunc: func(ctx context.Context, merchantID, key string) (*domain.Transaction, error) {
			return existingTx, nil
		},
	}

	acqCalled := false
	acq := &MockAcquirerClient{
		AuthorizeFunc: func(ctx context.Context, authReq acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error) {
			acqCalled = true
			return nil, errors.New("should not be called")
		},
	}

	svc := processpayment.NewService(repo, acq, 10000000)
	out, err := svc.ProcessPayment(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, out.IsReplayed)
	assert.Equal(t, "APPROVED", out.Status)
	assert.Equal(t, "AUT-PREV", out.AuthorizationCode)
	assert.False(t, acqCalled, "acquirer must not be called on replay")
}

// 6. Idempotency Conflict (reusing key with different amount or card)
func TestService_ProcessPayment_Idempotent_Conflict(t *testing.T) {
	req := validSampleRequest()
	differentHash := "different-sha256-hash"
	now := time.Now().UTC()

	existingTx := domain.NewTransaction(uuid.New(), uuid.New(), req.MerchantID, req.IdempotencyKey, differentHash, "424242", "4242", "corr-prev", now)

	repo := &MockRepository{
		GetTransactionByIdempotencyKeyFunc: func(ctx context.Context, merchantID, key string) (*domain.Transaction, error) {
			return existingTx, nil
		},
	}

	svc := processpayment.NewService(repo, &MockAcquirerClient{}, 10000000)
	_, err := svc.ProcessPayment(context.Background(), req)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

// 7. Active Transaction in Flight (< 5min)
func TestService_ProcessPayment_ActiveTransaction_Collision(t *testing.T) {
	req := validSampleRequest()
	now := time.Now().UTC()
	hash := domain.ComputePayloadHash(req.MerchantID, req.OrderReference, req.Amount, req.Currency, "424242", "4242", req.CardExpiryMonth, req.CardExpiryYear)

	existingTx := domain.NewTransaction(uuid.New(), uuid.New(), req.MerchantID, req.IdempotencyKey, hash, "424242", "4242", "corr-prev", now.Add(-1*time.Minute))
	_ = existingTx.TransitionTo(domain.StatusProcessing, now.Add(-1*time.Minute))

	repo := &MockRepository{
		GetTransactionByIdempotencyKeyFunc: func(ctx context.Context, merchantID, key string) (*domain.Transaction, error) {
			return existingTx, nil
		},
	}

	svc := processpayment.NewService(repo, &MockAcquirerClient{}, 10000000)
	_, err := svc.ProcessPayment(context.Background(), req)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrActiveTransactionExists)
}

// 8. Acquirer Unknown Result (keeps PROCESSING)
func TestService_ProcessPayment_AcquirerUnknownResult(t *testing.T) {
	repo := &MockRepository{}
	acq := &MockAcquirerClient{
		AuthorizeFunc: func(ctx context.Context, req acquirer.AuthorizeRequest) (*acquirer.AcquirerResponse, error) {
			return nil, domain.ErrAcquirerUnknownResult
		},
	}

	svc := processpayment.NewService(repo, acq, 10000000)
	out, err := svc.ProcessPayment(context.Background(), validSampleRequest())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAcquirerUnknownResult)
	assert.Equal(t, "PROCESSING", out.Status)
}
