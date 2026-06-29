package observability

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
)

const codeAttrsCallersSkip = 5

// codeAttrsHandler injects OTEL/Sentry code attributes before forwarding to Sentry.
type codeAttrsHandler struct {
	next slog.Handler
}

func newCodeAttrsHandler(next slog.Handler) slog.Handler {
	return &codeAttrsHandler{next: next}
}

func (h *codeAttrsHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *codeAttrsHandler) Handle(ctx context.Context, r slog.Record) error {
	next := h.next
	if attrs := codeAttrsFromCallers(); len(attrs) > 0 {
		next = next.WithAttrs(attrs)
	}
	if svc := CurrentService(); svc != "" {
		next = next.WithAttrs([]slog.Attr{slog.String("service", svc)})
	}
	return next.Handle(ctx, r)
}

func (h *codeAttrsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &codeAttrsHandler{next: h.next.WithAttrs(attrs)}
}

func (h *codeAttrsHandler) WithGroup(name string) slog.Handler {
	return &codeAttrsHandler{next: h.next.WithGroup(name)}
}

func codeAttrsFromCallers() []slog.Attr {
	var pcs [32]uintptr
	n := runtime.Callers(codeAttrsCallersSkip, pcs[:])
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if shouldSkipCodeFrame(frame) {
			if !more {
				break
			}
			continue
		}
		return attrsFromFrame(frame)
	}
	return nil
}

func shouldSkipCodeFrame(frame runtime.Frame) bool {
	if frame.Function == "" && frame.File == "" {
		return true
	}
	if strings.Contains(frame.File, "/utils/observability/") {
		return true
	}
	if strings.Contains(frame.Function, "log/slog") {
		return true
	}
	return false
}

func attrsFromFrame(frame runtime.Frame) []slog.Attr {
	var attrs []slog.Attr
	if frame.Function != "" {
		attrs = append(attrs,
			slog.String("code.module.name", moduleNameFromFunction(frame.Function)),
			slog.String("code.function", functionNameFromFunction(frame.Function)),
		)
	}
	if frame.File != "" {
		attrs = append(attrs, slog.String("code.file.path", frame.File))
		if svc := serviceFromFilePath(frame.File); svc != "" {
			attrs = append(attrs, slog.String("origin.service", svc))
		}
	}
	if frame.Line > 0 {
		attrs = append(attrs, slog.Int("code.line.number", frame.Line))
	}
	return attrs
}

func serviceFromFilePath(file string) string {
	const marker = "/services/"
	idx := strings.Index(file, marker)
	if idx == -1 {
		return ""
	}
	rest := file[idx+len(marker):]
	if end := strings.Index(rest, "/"); end != -1 {
		return rest[:end]
	}
	return rest
}

func moduleNameFromFunction(fn string) string {
	if fn == "" {
		return ""
	}

	lastDot := strings.LastIndex(fn, ".")
	if lastDot == -1 {
		return fn
	}

	pkg := fn[:lastDot]
	if paren := strings.LastIndex(pkg, "("); paren != -1 {
		if dot := strings.LastIndex(pkg[:paren], "."); dot != -1 {
			return pkg[:dot]
		}
		return pkg[:paren]
	}

	return pkg
}

func functionNameFromFunction(fn string) string {
	if fn == "" {
		return ""
	}

	lastDot := strings.LastIndex(fn, ".")
	if lastDot == -1 || lastDot == len(fn)-1 {
		return fn
	}

	return fn[lastDot+1:]
}
