package observability

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// RequestLogMiddleware logs every HTTP request with method, path, status, and duration.
func RequestLogMiddleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			logger := RequestLog
			if logger == nil {
				logger = DefaultLog
			}
			if logger == nil {
				return
			}

			reqID := middleware.GetReqID(r.Context())
			attrs := []any{
				"service", service,
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
			}
			if reqID != "" {
				attrs = append(attrs, "request_id", reqID)
			}
			if q := r.URL.RawQuery; q != "" {
				attrs = append(attrs, "query", q)
			}

			status := ww.Status()
			switch {
			case status >= 500:
				logger.ErrorContext(r.Context(), "http request", attrs...)
			case status >= 400:
				logger.WarnContext(r.Context(), "http request", attrs...)
			default:
				logger.InfoContext(r.Context(), "http request", attrs...)
			}
		})
	}
}
