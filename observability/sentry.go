package observability

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
)

// InitSentry initializes Sentry when SENTRY_DSN is set; no-op otherwise.
func InitSentry(serviceName string) {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return
	}

	sampleRate := 0.1
	if v := os.Getenv("SENTRY_TRACES_SAMPLE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			sampleRate = f
		}
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      envOr("SENTRY_ENVIRONMENT", "development"),
		ServerName:       serviceName,
		TracesSampleRate: sampleRate,
		EnableTracing:    true,
	}); err != nil {
		log.Printf("sentry init failed for %s: %v", serviceName, err)
		return
	}
	log.Printf("sentry initialized for %s (structured logs enabled)", serviceName)
}

// FlushSentry flushes pending Sentry events.
func FlushSentry(timeout time.Duration) {
	if os.Getenv("SENTRY_DSN") == "" {
		return
	}
	sentry.Flush(timeout)
}

// CaptureError reports an error to Sentry when enabled.
func CaptureError(err error, tags map[string]string) {
	if err == nil || os.Getenv("SENTRY_DSN") == "" {
		return
	}
	hub := sentry.CurrentHub().Clone()
	for k, v := range tags {
		hub.Scope().SetTag(k, v)
	}
	hub.CaptureException(err)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
