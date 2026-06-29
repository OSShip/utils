package observability

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	sentryslog "github.com/getsentry/sentry-go/slog"
	"github.com/go-chi/chi/v5/middleware"
)

// DefaultLog is the service-scoped logger set by InitLogger.
var DefaultLog *slog.Logger

// RequestLog writes HTTP access logs to a file (never to Sentry).
var RequestLog *slog.Logger

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
			AddSource:  false,
			AttrFromContext: []func(context.Context) []slog.Attr{
				requestIDFromContext,
			},
		}.NewSentryHandler(context.Background())
		sentryHandler = newCodeAttrsHandler(excludeHTTPRequestLogs(sentryHandler))
		handlers = append(handlers, sentryHandler)
	}

	logger := slog.New(newMultiHandler(handlers...)).With("service", service)
	setCurrentService(service)
	DefaultLog = logger
	RequestLog = initRequestLogger(service, level)
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

func initRequestLogger(service string, level slog.Level) *slog.Logger {
	path := os.Getenv("HTTP_REQUEST_LOG_FILE")
	if path == "" {
		path = filepath.Join("/logs", fmt.Sprintf("%s-http-requests.log", service))
		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			path = filepath.Join("logs", fmt.Sprintf("%s-http-requests.log", service))
		}
	}

	opts := &slog.HandlerOptions{Level: level}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("request log: mkdir %q failed: %v", filepath.Dir(path), err)
		return slog.New(slog.NewTextHandler(os.Stdout, opts)).With("service", service)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("request log: open %q failed: %v", path, err)
		return slog.New(slog.NewTextHandler(os.Stdout, opts)).With("service", service)
	}

	fileHandler := slog.NewTextHandler(f, opts)
	logger := slog.New(fileHandler).With("service", service)
	logger.Info("http request logging enabled", "file", path)
	return logger
}

// excludeHTTPRequestLogs prevents access-log noise from reaching Sentry.
func excludeHTTPRequestLogs(next slog.Handler) slog.Handler {
	return &httpRequestLogFilter{next: next}
}

type httpRequestLogFilter struct {
	next slog.Handler
}

func (f *httpRequestLogFilter) Enabled(ctx context.Context, level slog.Level) bool {
	return f.next.Enabled(ctx, level)
}

func (f *httpRequestLogFilter) Handle(ctx context.Context, r slog.Record) error {
	if r.Message == "http request" {
		return nil
	}
	return f.next.Handle(ctx, r)
}

func (f *httpRequestLogFilter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &httpRequestLogFilter{next: f.next.WithAttrs(attrs)}
}

func (f *httpRequestLogFilter) WithGroup(name string) slog.Handler {
	return &httpRequestLogFilter{next: f.next.WithGroup(name)}
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
