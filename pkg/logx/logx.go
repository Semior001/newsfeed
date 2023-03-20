// Package logx contains logging middleware for request id.
package logx

import (
	"context"

	"golang.org/x/exp/slog"
)

// HandleFunc is a function that handles a record.
type HandleFunc func(context.Context, slog.Record) error

// Middleware is a middleware for logging handler.
type Middleware func(HandleFunc) HandleFunc

// Chain is a chain of middleware.
type Chain struct {
	Middleware []Middleware
	slog.Handler
}

// Handle runs the chain of middleware and the handler.
func (c *Chain) Handle(ctx context.Context, rec slog.Record) error {
	h := c.Handler.Handle
	for i := len(c.Middleware) - 1; i >= 0; i-- {
		h = c.Middleware[i](h)
	}
	return h(ctx, rec)
}

// WithGroup returns a new Chain with the given group.
func (c *Chain) WithGroup(group string) slog.Handler {
	return &Chain{
		Middleware: c.Middleware,
		Handler:    c.Handler.WithGroup(group),
	}
}

// WithAttrs returns a new Chain with the given attributes.
func (c *Chain) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Chain{
		Middleware: c.Middleware,
		Handler:    c.Handler.WithAttrs(attrs),
	}
}
