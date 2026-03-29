// Package loghandler contains loggers middleware
package loghandler

import (
	"context"
	"log/slog"
	"runtime"
)

type handlerMiddlware struct {
	next slog.Handler
}

// NewHandlerMiddleware - constructor for slog middleware
func NewHandlerMiddleware(next slog.Handler) *handlerMiddlware {
	return &handlerMiddlware{next: next}
}

// Enabled - control levels of logging
func (h *handlerMiddlware) Enabled(ctx context.Context, rec slog.Level) bool {
	return h.next.Enabled(ctx, rec)
}

// Handle - log handler
func (h *handlerMiddlware) Handle(ctx context.Context, rec slog.Record) error {
	data, ok := GetData(ctx)
	if ok {
		for k, v := range data {
			rec.Add(k, v)
		}
	}

	if IsWithSource(ctx) {
		if pc, file, line, ok := runtime.Caller(3); ok {
			fn := runtime.FuncForPC(pc).Name()

			rec.AddAttrs(
				slog.Group("source",
					slog.String("file", file),
					slog.String("function", fn),
					slog.Int("line", line),
				),
			)
		}
	}

	return h.next.Handle(ctx, rec)
}

// WithAttrs - add attrs
func (h *handlerMiddlware) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handlerMiddlware{next: h.next.WithAttrs(attrs)}
}

// WithGroup - add group
func (h *handlerMiddlware) WithGroup(name string) slog.Handler {
	return &handlerMiddlware{next: h.next.WithGroup(name)}
}
