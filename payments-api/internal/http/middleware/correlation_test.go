package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/http/middleware"
)

func TestCorrelationMiddleware_ExistingHeader(t *testing.T) {
	handler := middleware.Correlation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := middleware.GetCorrelationID(r.Context())
		assert.Equal(t, "custom-corr-id-123", corrID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Correlation-ID", "custom-corr-id-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "custom-corr-id-123", res.Header.Get("X-Correlation-ID"))
}

func TestCorrelationMiddleware_GeneratedIfMissing(t *testing.T) {
	var capturedID string
	handler := middleware.Correlation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = middleware.GetCorrelationID(r.Context())
		assert.NotEmpty(t, capturedID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, capturedID, res.Header.Get("X-Correlation-ID"))
}
