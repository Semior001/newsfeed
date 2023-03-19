// Package route contains definitions for routing and handling requests.
package route

import "context"

// Handler handles requests.
type Handler func(ctx context.Context, req Request) ([]Response, error)

// Response is a response from handler.
type Response struct {
	ChatID string
	Text   string
}

// Request is a request for handler.
type Request struct {
	Chat Chat
	Text string
}

// Chat contains chat information.
type Chat struct {
	ID       string
	Username string
}

// NotFound is a default handler for not found commands.
func NotFound(_ context.Context, req Request) ([]Response, error) {
	return []Response{{
		ChatID: req.Chat.ID,
		Text:   "command not found",
	}}, nil
}