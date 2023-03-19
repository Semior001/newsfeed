package route

import (
	"context"
	"fmt"

	"github.com/Semior001/newsfeed/app/logging"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

// Logger is a middleware that logs all requests
func Logger(lg *slog.Logger) func(Handler) Handler {
	return func(next Handler) Handler {
		return func(ctx context.Context, req Request) ([]Response, error) {
			args := []any{
				slog.String("chat_id", req.Chat.ID),
				slog.String("chat_username", req.Chat.Username),
			}

			if lg.Handler().Enabled(ctx, slog.LevelDebug) {
				lg.DebugCtx(ctx, "request received", append(args, slog.String("command", req.Text))...)
			} else {
				lg.InfoCtx(ctx, "request received", args...)
			}

			res, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			if lg.Handler().Enabled(ctx, slog.LevelDebug) {
				lg.DebugCtx(ctx, "request processed", slog.Any("responses", res))
			} else {
				lg.InfoCtx(ctx, "request processed",
					slog.Any("responses", lo.Map(res, func(r Response, _ int) Response {
						return Response{ChatID: r.ChatID}
					})),
				)
			}

			return res, nil
		}
	}
}

// Recover is a middleware that recovers from panics.
func Recover(lg *slog.Logger) func(Handler) Handler {
	return func(next Handler) Handler {
		return func(ctx context.Context, req Request) ([]Response, error) {
			defer func() {
				if r := recover(); r != nil {
					lg.ErrorCtx(ctx, "panic recovered", slog.Any("panic", r))
				}
			}()

			return next(ctx, req)
		}
	}
}

// RequestID is a middleware that adds request id to context.
func RequestID(appendID bool) func(next Handler) Handler {
	return func(next Handler) Handler {
		return func(ctx context.Context, req Request) ([]Response, error) {
			id := uuid.New().String()
			ctx = logging.ContextWithRequestID(ctx, id)

			resps, err := next(ctx, req)

			if appendID {
				if reqID, ok := logging.RequestIDFromContext(ctx); ok {
					for i := range resps {
						resps[i].Text += fmt.Sprintf("\n\nRequest ID: %s", reqID)
					}
				}
			}

			return resps, err
		}
	}
}
