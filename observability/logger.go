package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"

	sentryslog "github.com/getsentry/sentry-go/slog"
	"github.com/go-chi/chi/v5/middleware"
)

// DefaultLog is the service-scoped logger set by InitLogger.
var DefaultLog *slog.Logger

// InitLogger configures structured logging to stdout and Sentry (when SENTRY_DSN is set).
// Call after InitSentry. Returns the logger for convenience.
func InitLogger(service string) *slog.Logger {
	level := parseLogLevel(os.Getenv("SENTRY_LOG_LEVEL"))

	stdout := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})

	handlers := []slog.Handler{stdout}

	if os.Getenv("SENTRY_DSN") != "" {
		sentryHandler := sentryslog.Option{
			EventLevel: []slog.Level{slog.LevelError, sentryslog.LevelFatal},
			LogLevel:   []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn},
			AddSource:  true,
			AttrFromContext: []func(context.Context) []slog.Attr{
				requestIDFromContext,
			},
		}.NewSentryHandler(context.Background())
		handlers = append(handlers, sentryHandler)
	}

	logger := slog.New(newMultiHandler(handlers...)).With("service", service)
	DefaultLog = logger
	slog.SetDefault(logger)

	if os.Getenv("SENTRY_DSN") != "" {
		logger.Info("logger initialized with Sentry structured logs", "min_level", level.String())
	} else {
		logger.Info("logger initialized (stdout only, Sentry disabled)")
	}

	return logger
}

func parseLogLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func requestIDFromContext(ctx context.Context) []slog.Attr {
	if id := middleware.GetReqID(ctx); id != "" {
		return []slog.Attr{slog.String("request_id", id)}
	}
	return nil
}

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			_ = h.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		next[i] = h.WithAttrs(attrs)
	}
	return newMultiHandler(next...)
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		next[i] = h.WithGroup(name)
	}
	return newMultiHandler(next...)
}
