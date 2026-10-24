package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterInterceptor) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger logs each HTTP request and its response status and latency with slog.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			interceptor := &responseWriterInterceptor{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default if WriteHeader is not called explicitly
			}

			next.ServeHTTP(interceptor, r)

			duration := time.Since(start)
			corrID := GetCorrelationID(r.Context())

			level := slog.LevelInfo
			if interceptor.statusCode >= 500 {
				level = slog.LevelError
			} else if interceptor.statusCode >= 400 {
				level = slog.LevelWarn
			}

			logger.Log(r.Context(), level, "HTTP request handled",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", interceptor.statusCode),
				slog.Duration("duration", duration),
				slog.String("correlation_id", corrID),
				slog.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}
