package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/http/middleware"
)

func TestRecoveryMiddleware(t *testing.T) {
	panickingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went completely haywire!")
	})

	handler := middleware.Recovery(panickingHandler)

	req := httptest.NewRequest(http.MethodGet, "/test-panic", nil)
	rec := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		handler.ServeHTTP(rec, req)
	})

	res := rec.Result()
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, "application/problem+json", res.Header.Get("Content-Type"))

	var problem struct {
		Status int    `json:"status"`
		Code   string `json:"code"`
	}
	err := json.NewDecoder(res.Body).Decode(&problem)
	require.NoError(t, err)

	assert.Equal(t, 500, problem.Status)
	assert.Equal(t, "internal_server_error", problem.Code)
}
