package route

import (
	"context"
	"strings"
)

type router struct {
	handlers map[string]Handler
}

// Router returns a multiplexer for handlers.
// "" is a reserved key for default handler.
func Router(handlers map[string]Handler) Handler {
	return (&router{handlers: handlers}).Handle
}

// Handle handles request.
func (r *router) Handle(ctx context.Context, req Request) ([]Response, error) {
	if req.Text == "" {
		return nil, nil
	}

	for prefix, h := range r.handlers {
		if prefix == "" {
			continue
		}
		if strings.HasPrefix(req.Text, prefix) {
			return h(ctx, req)
		}
	}

	h, ok := r.handlers[""]
	if !ok {
		return NotFound(ctx, req)
	}

	return h(ctx, req)
}
