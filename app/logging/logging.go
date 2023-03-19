package logging

import (
	"context"

	"golang.org/x/exp/slog"
)

type requestIDKey struct{}

// ContextWithRequestID returns a new context with the given request ID.
func ContextWithRequestID(parent context.Context, reqID string) context.Context {
	return context.WithValue(parent, requestIDKey{}, reqID)
}

// RequestIDFromContext returns request id from context.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDKey{}).(string)
	return v, ok
}

// Handler is a middleware for logging request id.
type Handler struct {
	slog.Handler
}

// Handle implements slog.Handler interface.
func (h Handler) Handle(ctx context.Context, rec slog.Record) error {
	if reqID, ok := RequestIDFromContext(ctx); ok {
		rec.AddAttrs(slog.String("request_id", reqID))
	}
	return h.Handler.Handle(ctx, rec)
}

// WithGroup returns a new Handler with the given group.
func (h Handler) WithGroup(group string) slog.Handler {
	return Handler{Handler: h.Handler.WithGroup(group)}
}

// WithAttrs returns a new Handler with the given attributes.
func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return Handler{Handler: h.Handler.WithAttrs(attrs)}
}
