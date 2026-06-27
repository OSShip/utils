package observability

import (
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"
)

// SentryRecoverMiddleware captures panics and reports them to Sentry.
func SentryRecoverMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					err := fmt.Errorf("panic in %s: %v", serviceName, rec)
					CaptureError(err, map[string]string{
						"service": serviceName,
						"path":    r.URL.Path,
						"method":  r.Method,
					})
					http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SentryErrorMiddleware captures 5xx responses as Sentry messages.
func SentryErrorMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			if rec.status >= 500 {
				CaptureError(fmt.Errorf("%s %s returned %d", r.Method, r.URL.Path, rec.status), map[string]string{
					"service": serviceName,
					"path":    r.URL.Path,
					"method":  r.Method,
					"status":  fmt.Sprintf("%d", rec.status),
				})
			}
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Ensure sentry hub is linked for request-scoped context when DSN is set.
func SentryHTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hub := sentry.GetHubFromContext(r.Context()); hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx := sentry.SetHubOnContext(r.Context(), hub)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
