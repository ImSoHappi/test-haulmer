package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

type problemResponse struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	Status        int    `json:"status"`
	Code          string `json:"code"`
	Detail        string `json:"detail"`
	Instance      string `json:"instance,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// Recovery catches panics from downstream handlers, logs the stack trace,
// and returns a clean RFC 9457 500 Internal Server Error.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				corrID := GetCorrelationID(r.Context())
				stack := string(debug.Stack())

				slog.Error("panic recovered in HTTP handler",
					slog.Any("error", rec),
					slog.String("correlation_id", corrID),
					slog.String("path", r.URL.Path),
					slog.String("stack", stack),
				)

				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusInternalServerError)

				_ = json.NewEncoder(w).Encode(problemResponse{
					Type:          "https://api.haulmer.com/errors/internal_server_error",
					Title:         "Internal Server Error",
					Status:        http.StatusInternalServerError,
					Code:          "internal_server_error",
					Detail:        "An unexpected error occurred while processing the request.",
					Instance:      r.URL.Path,
					CorrelationID: corrID,
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
