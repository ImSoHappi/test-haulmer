package http_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	apihttp "github.com/ImSoHappi/test-haulmer/payments-api/internal/http"
)

func TestNewRouter_RegistersRoutes(t *testing.T) {
	logger := slog.Default()
	handler := apihttp.NewPaymentHandler(nil)
	router := apihttp.NewRouter(logger, handler)

	// Test route not found
	req := httptest.NewRequest(http.MethodGet, "/unknown-path", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-Correlation-ID"), "must inject correlation id on any request")

	// Test POST /v1/payments with missing idempotency key (validates route exists and hits handler)
	reqPost := httptest.NewRequest(http.MethodPost, "/v1/payments", nil)
	recPost := httptest.NewRecorder()
	router.ServeHTTP(recPost, reqPost)

	assert.Equal(t, http.StatusBadRequest, recPost.Code)
}
