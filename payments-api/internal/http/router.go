package http

import (
	"log/slog"
	"net/http"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/http/middleware"
)

// NewRouter builds and returns an http.Handler configured with net/http.ServeMux and global middlewares.
func NewRouter(logger *slog.Logger, paymentHandler *PaymentHandler) http.Handler {
	mux := http.NewServeMux()

	// Health Check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP"}`))
	})

	// Business Routes
	mux.HandleFunc("POST /v1/payments", paymentHandler.ProcessPayment)
	mux.HandleFunc("GET /v1/payments/{id}", paymentHandler.GetPayment)
	mux.HandleFunc("GET /v1/payments", paymentHandler.ListPayments)

	// Apply Middlewares in order:
	// 1. Recovery (outermost: catches any panic)
	// 2. Correlation (assigns X-Correlation-ID to ctx and response)
	// 3. Logger (logs request, status, duration and correlation id)
	var handler http.Handler = mux
	handler = middleware.Logger(logger)(handler)
	handler = middleware.Correlation(handler)
	handler = middleware.Recovery(handler)

	return handler
}
