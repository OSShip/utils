package observability

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// RespondError logs the failure (stdout + Sentry for 5xx), then writes a JSON error body.
// operation describes what failed (e.g. "create session"); publicMessage is returned to the client.
func RespondError(w http.ResponseWriter, r *http.Request, status int, publicMessage, operation string, err error, attrs ...any) {
	log := DefaultLog
	if log == nil {
		log = slog.Default()
	}

	args := make([]any, 0, len(attrs)+10)
	args = append(args,
		"status", status,
		"operation", operation,
		"public_error", publicMessage,
		"method", r.Method,
		"path", r.URL.Path,
	)
	if svc := CurrentService(); svc != "" {
		args = append(args, "service", svc)
	}
	if reqID := middleware.GetReqID(r.Context()); reqID != "" {
		args = append(args, "request_id", reqID)
	}
	if err != nil {
		args = append(args, "err", err)
	}
	args = append(args, attrs...)

	switch {
	case status >= http.StatusInternalServerError:
		log.ErrorContext(r.Context(), "request failed", args...)
		captureRequestError(r, operation, publicMessage, err)
	case status >= http.StatusBadRequest:
		log.WarnContext(r.Context(), "request failed", args...)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": publicMessage})
}

func captureRequestError(r *http.Request, operation, publicMessage string, err error) {
	tags := map[string]string{
		"operation":    operation,
		"public_error": publicMessage,
		"path":         r.URL.Path,
		"method":       r.Method,
	}
	if err == nil {
		err = fmt.Errorf("%s: %s", operation, publicMessage)
	}
	CaptureError(err, tags)
}
