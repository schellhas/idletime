package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/schellhas/idletime/backend/internal/httpx"
)

type middleware func(http.Handler) http.Handler

// statusRecorder captures the status code written by downstream handlers.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func requestLogger(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration", time.Since(start),
			)
		})
	}
}

func recoverer(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic recovered", "panic", recovered)
					httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
