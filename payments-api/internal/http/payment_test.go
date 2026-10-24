package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
	apihttp "github.com/ImSoHappi/test-haulmer/payments-api/internal/http"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/payment"
)

type mockPaymentService struct {
	processPaymentFn func(ctx context.Context, req payment.ProcessRequest) (*payment.ProcessResponse, error)
	getPaymentFn     func(ctx context.Context, req payment.GetRequest) (*payment.GetResponse, error)
	listPaymentsFn   func(ctx context.Context, req payment.ListRequest) (*payment.ListResponse, error)
}

func (m *mockPaymentService) ProcessPayment(ctx context.Context, req payment.ProcessRequest) (*payment.ProcessResponse, error) {
	if m.processPaymentFn != nil {
		return m.processPaymentFn(ctx, req)
	}
	return nil, nil
}

func (m *mockPaymentService) GetPayment(ctx context.Context, req payment.GetRequest) (*payment.GetResponse, error) {
	if m.getPaymentFn != nil {
		return m.getPaymentFn(ctx, req)
	}
	return nil, nil
}

func (m *mockPaymentService) ListPayments(ctx context.Context, req payment.ListRequest) (*payment.ListResponse, error) {
	if m.listPaymentsFn != nil {
		return m.listPaymentsFn(ctx, req)
	}
	return nil, nil
}

func validPaymentPayload() map[string]any {
	return map[string]any{
		"merchant_id":       "1001",
		"order_reference":   "KD0877",
		"amount":            25000,
		"currency":          "CLP",
		"card_number":       "4242424242424242",
		"card_expiry_month": 12,
		"card_expiry_year":  2030,
		"card_cvv":          "123",
		"card_holder_name":  "Ada Lovelace",
	}
}

func TestPaymentHandler_ProcessPayment_Success(t *testing.T) {
	txID := uuid.New()
	piID := uuid.New()
	now := time.Now().UTC()

	mockSvc := &mockPaymentService{
		processPaymentFn: func(ctx context.Context, req payment.ProcessRequest) (*payment.ProcessResponse, error) {
			assert.Equal(t, "1001", req.MerchantID)
			assert.Equal(t, "idem-test-key-1", req.IdempotencyKey)
			assert.Equal(t, "KD0877", req.OrderReference)
			assert.Equal(t, int64(25000), req.Amount)
			return &payment.ProcessResponse{
				TransactionID:     txID.String(),
				PaymentIntentID:   piID.String(),
				MerchantID:        req.MerchantID,
				OrderReference:    req.OrderReference,
				Amount:            req.Amount,
				Currency:          req.Currency,
				Status:            "APPROVED",
				CardBIN:           "424242",
				CardLast4:         "4242",
				AuthorizationCode: "AUTH-01",
				CreatedAt:         now,
			}, nil
		},
	}

	handler := apihttp.NewPaymentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/payments", handler.ProcessPayment)

	bodyBytes, err := json.Marshal(validPaymentPayload())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/payments", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-test-key-1")
	req.Header.Set("X-Correlation-ID", "corr-test-01")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var respBody payment.ProcessResponse
	err = json.Unmarshal(rec.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.Equal(t, txID.String(), respBody.TransactionID)
	assert.Equal(t, "APPROVED", respBody.Status)
}

func TestPaymentHandler_ProcessPayment_IdempotentReplay(t *testing.T) {
	txID := uuid.New()
	piID := uuid.New()
	now := time.Now().UTC()

	mockSvc := &mockPaymentService{
		processPaymentFn: func(ctx context.Context, req payment.ProcessRequest) (*payment.ProcessResponse, error) {
			return &payment.ProcessResponse{
				TransactionID:     txID.String(),
				PaymentIntentID:   piID.String(),
				MerchantID:        req.MerchantID,
				OrderReference:    req.OrderReference,
				Amount:            req.Amount,
				Currency:          req.Currency,
				Status:            "APPROVED",
				CardBIN:           "424242",
				CardLast4:         "4242",
				AuthorizationCode: "AUTH-01",
				IsReplayed:        true,
				CreatedAt:         now,
			}, nil
		},
	}

	handler := apihttp.NewPaymentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/payments", handler.ProcessPayment)

	bodyBytes, err := json.Marshal(validPaymentPayload())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/payments", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-test-key-replay")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "true", rec.Header().Get("Idempotent-Replayed"))

	var respBody payment.ProcessResponse
	err = json.Unmarshal(rec.Body.Bytes(), &respBody)
	require.NoError(t, err)
	assert.True(t, respBody.IsReplayed)
}

func TestPaymentHandler_ProcessPayment_Declined_402(t *testing.T) {
	txID := uuid.New()
	piID := uuid.New()
	now := time.Now().UTC()

	mockSvc := &mockPaymentService{
		processPaymentFn: func(ctx context.Context, req payment.ProcessRequest) (*payment.ProcessResponse, error) {
			return &payment.ProcessResponse{
				TransactionID:   txID.String(),
				PaymentIntentID: piID.String(),
				MerchantID:      req.MerchantID,
				OrderReference:  req.OrderReference,
				Amount:          req.Amount,
				Currency:        req.Currency,
				Status:          "DECLINED",
				DeclineSource:   "acquirer",
				ResultCode:      "51",
				ResultMessage:   "Insufficient funds",
				CardBIN:         "424242",
				CardLast4:       "4242",
				CreatedAt:       now,
			}, nil
		},
	}

	handler := apihttp.NewPaymentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/payments", handler.ProcessPayment)

	bodyBytes, err := json.Marshal(validPaymentPayload())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/payments", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-declined")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusPaymentRequired, rec.Code)
}

func TestPaymentHandler_ProcessPayment_MissingIdempotencyKey(t *testing.T) {
	mockSvc := &mockPaymentService{}
	handler := apihttp.NewPaymentHandler(mockSvc)

	bodyBytes, err := json.Marshal(validPaymentPayload())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/payments", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	handler.ProcessPayment(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
}

func TestPaymentHandler_ProcessPayment_DomainErrorRFC9457(t *testing.T) {
	mockSvc := &mockPaymentService{
		processPaymentFn: func(ctx context.Context, req payment.ProcessRequest) (*payment.ProcessResponse, error) {
			return nil, domain.ErrMerchantInactive
		},
	}

	handler := apihttp.NewPaymentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/payments", handler.ProcessPayment)

	bodyBytes, err := json.Marshal(validPaymentPayload())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/payments", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-inactive")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var errResp map[string]any
	err = json.Unmarshal(rec.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "merchant_inactive", errResp["code"])
}

func TestPaymentHandler_GetPayment_Success(t *testing.T) {
	txID := uuid.New()
	mockSvc := &mockPaymentService{
		getPaymentFn: func(ctx context.Context, req payment.GetRequest) (*payment.GetResponse, error) {
			assert.Equal(t, txID.String(), req.TransactionID)
			assert.Equal(t, "1001", req.MerchantID)
			return &payment.GetResponse{
				ID:         txID.String(),
				MerchantID: req.MerchantID,
				Status:     "APPROVED",
				Events:     []payment.EventItem{},
			}, nil
		},
	}

	handler := apihttp.NewPaymentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/payments/{id}", handler.GetPayment)

	req := httptest.NewRequest(http.MethodGet, "/v1/payments/"+txID.String(), nil)
	req.Header.Set("X-Merchant-ID", "1001")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp payment.GetResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, txID.String(), resp.ID)
}

func TestPaymentHandler_GetPayment_InvalidUUID(t *testing.T) {
	mockSvc := &mockPaymentService{
		getPaymentFn: func(ctx context.Context, req payment.GetRequest) (*payment.GetResponse, error) {
			return nil, nil
		},
	}

	handler := apihttp.NewPaymentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/payments/{id}", handler.GetPayment)

	req := httptest.NewRequest(http.MethodGet, "/v1/payments/not-a-valid-uuid", nil)
	req.Header.Set("X-Merchant-ID", "1001")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPaymentHandler_ListPayments_Success(t *testing.T) {
	txID := uuid.New()
	mockSvc := &mockPaymentService{
		listPaymentsFn: func(ctx context.Context, req payment.ListRequest) (*payment.ListResponse, error) {
			assert.Equal(t, "1001", req.MerchantID)
			assert.Equal(t, 10, req.Limit)
			assert.Equal(t, 20, req.Offset)
			return &payment.ListResponse{
				Items: []payment.ListItem{
					{ID: txID.String(), MerchantID: req.MerchantID, Status: "APPROVED"},
				},
				TotalCount: 1,
				Limit:      req.Limit,
				Offset:     req.Offset,
			}, nil
		},
	}

	handler := apihttp.NewPaymentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/payments", handler.ListPayments)

	req := httptest.NewRequest(http.MethodGet, "/v1/payments?limit=10&offset=20", nil)
	req.Header.Set("X-Merchant-ID", "1001")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp payment.ListResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.TotalCount)
	assert.Len(t, resp.Items, 1)
}

func TestPaymentHandler_ListPayments_MissingMerchantHeader(t *testing.T) {
	mockSvc := &mockPaymentService{}
	handler := apihttp.NewPaymentHandler(mockSvc)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/payments", handler.ListPayments)

	req := httptest.NewRequest(http.MethodGet, "/v1/payments", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
