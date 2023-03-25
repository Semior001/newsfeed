package botx

import (
	"context"
	"strings"
)

// Router returns a multiplexer for handlers.
type Router struct {
	notFound    Handler
	handlers    map[string]Handler
	middlewares []Middleware
}

// NewRouter returns a multiplexer for handlers.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]Handler),
		notFound: NotFound,
	}
}

// Add adds a handler to the router.
func (r *Router) Add(prefix string, h Handler) {
	r.handlers[prefix] = h
}

// Use applies middleware to all handlers.
func (r *Router) Use(mvs ...Middleware) *Router {
	r.middlewares = append(r.middlewares, mvs...)
	return r
}

// With returns a new router with middleware applied.
func (r *Router) With(mvs ...Middleware) *Router {
	return r.Clone().Use(mvs...)
}

// Clone returns a copy of the router.
func (r *Router) Clone() *Router {
	rtr := NewRouter()

	rtr.handlers = make(map[string]Handler, len(r.handlers))
	for prefix, h := range r.handlers {
		rtr.Add(prefix, h)
	}

	rtr.middlewares = make([]Middleware, len(r.middlewares))
	copy(rtr.middlewares, r.middlewares)

	return rtr
}

// Group groups handlers.
func (r *Router) Group(f func(rtr *Router)) {
	nested := NewRouter()
	f(nested)

	for prefix, h := range nested.handlers {
		// wrap handler with middlewares
		for i := len(nested.middlewares) - 1; i >= 0; i-- {
			h = nested.middlewares[i](h)
		}
		r.Add(prefix, h)
	}
}

// NotFound sets a not found handler to the router.
func (r *Router) NotFound(h Handler) {
	r.notFound = h
}

// Handle handles request.
func (r *Router) Handle(ctx context.Context, req Request) ([]Response, error) {
	if req.Text == "" {
		return nil, nil
	}

	var h Handler

	for prefix, candidate := range r.handlers {
		if prefix == "" {
			continue
		}
		if strings.HasPrefix(req.Text, prefix) {
			h = candidate
			break
		}
	}

	if h == nil {
		h = r.notFound
	}

	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}

	return h(ctx, req)
}
