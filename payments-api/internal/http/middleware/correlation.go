package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type correlationKey struct{}

// CorrelationHeader is the standard HTTP header for request tracing.
const CorrelationHeader = "X-Correlation-ID"

// Correlation middleware extracts X-Correlation-ID from headers or generates a new UUID.
// It injects the correlation ID into context.Context and writes it back to the response headers.
func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := strings.TrimSpace(r.Header.Get(CorrelationHeader))
		if corrID == "" {
			corrID = uuid.NewString()
		}

		w.Header().Set(CorrelationHeader, corrID)
		ctx := context.WithValue(r.Context(), correlationKey{}, corrID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithCorrelationID manually injects a correlation ID into a context (useful for testing).
func WithCorrelationID(ctx context.Context, corrID string) context.Context {
	return context.WithValue(ctx, correlationKey{}, corrID)
}

// GetCorrelationID retrieves the correlation ID from context, or empty string if not found.
func GetCorrelationID(ctx context.Context) string {
	if val, ok := ctx.Value(correlationKey{}).(string); ok {
		return val
	}
	return ""
}
