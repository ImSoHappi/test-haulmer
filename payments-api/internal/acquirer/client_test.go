package acquirer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/acquirer"
	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

func sampleAuthorizeRequest(refID, corrID string) acquirer.AuthorizeRequest {
	return acquirer.AuthorizeRequest{
		ReferenceID:   refID,
		Amount:        19990,
		Currency:      "CLP",
		CorrelationID: corrID,
		Card: acquirer.CardRequest{
			Number:      "4242424242424242",
			ExpiryMonth: 12,
			ExpiryYear:  2028,
			CVV:         "123",
			HolderName:  "Test User",
		},
	}
}

// 1. Success Immediate
func TestClient_Authorize_SuccessImmediate(t *testing.T) {
	var requestCount int32
	var receivedCorrID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		receivedCorrID = r.Header.Get("X-Correlation-ID")

		assert.Equal(t, "/authorizations", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(acquirer.AcquirerResponse{
			ReferenceID:       "ref-123",
			Status:            "APPROVED",
			Code:              "00",
			AuthorizationCode: "AUT-998877",
			Amount:            19990,
			Currency:          "CLP",
			CreatedAt:         time.Now(),
		})
	}))
	defer server.Close()

	client := acquirer.NewClient(acquirer.ClientConfig{
		BaseURL:          server.URL,
		RequestTimeout:   1 * time.Second,
		ReconcileTimeout: 1 * time.Second,
		InitialBackoff:   10 * time.Millisecond,
		MaxRetries:       2,
	})

	resp, err := client.Authorize(context.Background(), sampleAuthorizeRequest("ref-123", "corr-xyz"))
	require.NoError(t, err)
	assert.Equal(t, "APPROVED", resp.Status)
	assert.Equal(t, "00", resp.Code)
	assert.Equal(t, "AUT-998877", resp.AuthorizationCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
	assert.Equal(t, "corr-xyz", receivedCorrID)
}

// 2. Decline Immediate (Never retries business errors: 51, 05, etc.)
func TestClient_Authorize_DeclineImmediate(t *testing.T) {
	testCases := []struct {
		name       string
		status     string
		code       string
		httpStatus int
	}{
		{"InsufficientFunds_51", "DECLINED", "51", http.StatusOK},
		{"DoNotHonor_05", "DECLINED", "05", http.StatusOK},
		{"ExpiredCard_54", "DECLINED", "54", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var requestCount int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&requestCount, 1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.httpStatus)
				_ = json.NewEncoder(w).Encode(acquirer.AcquirerResponse{
					ReferenceID: "ref-decline",
					Status:      tc.status,
					Code:        tc.code,
					Amount:      19990,
					Currency:    "CLP",
					CreatedAt:   time.Now(),
				})
			}))
			defer server.Close()

			client := acquirer.NewClient(acquirer.ClientConfig{
				BaseURL:          server.URL,
				RequestTimeout:   1 * time.Second,
				ReconcileTimeout: 1 * time.Second,
				InitialBackoff:   5 * time.Millisecond,
				MaxRetries:       3,
			})

			resp, err := client.Authorize(context.Background(), sampleAuthorizeRequest("ref-decline", "corr-decline"))
			require.NoError(t, err)
			assert.Equal(t, tc.status, resp.Status)
			assert.Equal(t, tc.code, resp.Code)
			// Must never retry business rejections: exactly 1 request sent
			assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
		})
	}
}

// 3. Client Errors (4xx) are Never Retried
func TestClient_Authorize_ClientError_NoRetry(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "malformed_request"})
	}))
	defer server.Close()

	client := acquirer.NewClient(acquirer.ClientConfig{
		BaseURL:          server.URL,
		RequestTimeout:   1 * time.Second,
		ReconcileTimeout: 1 * time.Second,
		InitialBackoff:   5 * time.Millisecond,
		MaxRetries:       3,
	})

	_, err := client.Authorize(context.Background(), sampleAuthorizeRequest("ref-bad", "corr-bad"))
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAcquirerUnexpected)
	// Must not retry on 400 Bad Request:
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
}

// 4. Retry on 503 with Eventual Success (Flaky endpoint)
func TestClient_Authorize_RetryOn503_EventualSuccess(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "temporary_system_overload"})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(acquirer.AcquirerResponse{
			ReferenceID:       "ref-retry",
			Status:            "APPROVED",
			Code:              "00",
			AuthorizationCode: "AUT-RETRY-OK",
			Amount:            19990,
			Currency:          "CLP",
			CreatedAt:         time.Now(),
		})
	}))
	defer server.Close()

	client := acquirer.NewClient(acquirer.ClientConfig{
		BaseURL:          server.URL,
		RequestTimeout:   1 * time.Second,
		ReconcileTimeout: 1 * time.Second,
		InitialBackoff:   5 * time.Millisecond,
		MaxRetries:       2, // Total attempts: 1 initial + 2 retries = 3
	})

	resp, err := client.Authorize(context.Background(), sampleAuthorizeRequest("ref-retry", "corr-retry"))
	require.NoError(t, err)
	assert.Equal(t, "APPROVED", resp.Status)
	assert.Equal(t, "AUT-RETRY-OK", resp.AuthorizationCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
}

// 5. Retry on 503 Exhausted
func TestClient_Authorize_RetryOn503_Exhausted(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "acquirer_down"})
	}))
	defer server.Close()

	client := acquirer.NewClient(acquirer.ClientConfig{
		BaseURL:          server.URL,
		RequestTimeout:   1 * time.Second,
		ReconcileTimeout: 1 * time.Second,
		InitialBackoff:   5 * time.Millisecond,
		MaxRetries:       2, // 1 initial + 2 retries = 3 calls
	})

	_, err := client.Authorize(context.Background(), sampleAuthorizeRequest("ref-down", "corr-down"))
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAcquirerUnavailable)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
}

// 6. Timeout with On-Demand Reconciliation Confirmed (APPROVED)
func TestClient_Authorize_Timeout_ReconciliationApproved(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Simulate slow acquirer execution exceeding request timeout
			time.Sleep(150 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/authorizations/ref-timeout-ok" {
			// Reconciliation confirms authorization exists
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(acquirer.AcquirerResponse{
				ReferenceID:       "ref-timeout-ok",
				Status:            "APPROVED",
				Code:              "00",
				AuthorizationCode: "AUT-RECON-OK",
				Amount:            19990,
				Currency:          "CLP",
				CreatedAt:         time.Now(),
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := acquirer.NewClient(acquirer.ClientConfig{
		BaseURL:          server.URL,
		RequestTimeout:   50 * time.Millisecond,
		ReconcileTimeout: 500 * time.Millisecond,
		InitialBackoff:   5 * time.Millisecond,
		MaxRetries:       0, // Go straight to reconciliation on timeout
	})

	resp, err := client.Authorize(context.Background(), sampleAuthorizeRequest("ref-timeout-ok", "corr-recon"))
	require.NoError(t, err)
	assert.Equal(t, "APPROVED", resp.Status)
	assert.Equal(t, "AUT-RECON-OK", resp.AuthorizationCode)
}

// 7. Timeout with On-Demand Reconciliation Not Found (FAILED)
func TestClient_Authorize_Timeout_ReconciliationNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			time.Sleep(150 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/authorizations/ref-timeout-404" {
			// Acquirer confirms no authorization was ever processed
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_not_found"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := acquirer.NewClient(acquirer.ClientConfig{
		BaseURL:          server.URL,
		RequestTimeout:   50 * time.Millisecond,
		ReconcileTimeout: 500 * time.Millisecond,
		InitialBackoff:   5 * time.Millisecond,
		MaxRetries:       0,
	})

	resp, err := client.Authorize(context.Background(), sampleAuthorizeRequest("ref-timeout-404", "corr-404"))
	require.NoError(t, err)
	assert.Equal(t, "FAILED", resp.Status)
	assert.Equal(t, "gateway_timeout", resp.Code)
}

// 8. Timeout with On-Demand Reconciliation Failed (returns ErrAcquirerUnknownResult)
func TestClient_Authorize_Timeout_ReconciliationFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			time.Sleep(150 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet {
			// Reconciliation itself also fails / times out
			time.Sleep(150 * time.Millisecond)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}))
	defer server.Close()

	client := acquirer.NewClient(acquirer.ClientConfig{
		BaseURL:          server.URL,
		RequestTimeout:   50 * time.Millisecond,
		ReconcileTimeout: 50 * time.Millisecond,
		InitialBackoff:   5 * time.Millisecond,
		MaxRetries:       0,
	})

	_, err := client.Authorize(context.Background(), sampleAuthorizeRequest("ref-unknown", "corr-unknown"))
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAcquirerUnknownResult)
}

// 9. Correlation ID Propagation
func TestClient_HeaderPropagation(t *testing.T) {
	var capturedHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("X-Correlation-ID")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(acquirer.AcquirerResponse{
			ReferenceID: "ref-header",
			Status:      "APPROVED",
			Code:        "00",
		})
	}))
	defer server.Close()

	client := acquirer.NewClient(acquirer.ClientConfig{
		BaseURL:          server.URL,
		RequestTimeout:   1 * time.Second,
		ReconcileTimeout: 1 * time.Second,
		InitialBackoff:   5 * time.Millisecond,
		MaxRetries:       0,
	})

	_, err := client.Authorize(context.Background(), sampleAuthorizeRequest("ref-header", "corr-unique-987"))
	require.NoError(t, err)
	assert.Equal(t, "corr-unique-987", capturedHeader)
}
