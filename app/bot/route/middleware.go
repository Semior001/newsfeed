package route

import (
	"context"
	"time"

	"golang.org/x/exp/slog"
)

// Logger is a middleware that logs all requests
func Logger(lg *slog.Logger) func(Handler) Handler {
	return func(next Handler) Handler {
		return func(ctx context.Context, req Request) ([]Response, error) {
			start := time.Now()

			res, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			args := []any{
				slog.String("chat_id", req.Chat.ID),
				slog.String("chat_username", req.Chat.Username),
				slog.Duration("duration", time.Since(start)),
			}

			if lg.Handler().Enabled(ctx, slog.LevelDebug) {
				lg.DebugCtx(ctx, "request processed", append(args, slog.String("command", req.Text))...)
			} else {
				lg.InfoCtx(ctx, "request processed", args...)
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

type requestIDKey struct{}

// RequestIDFromContext returns request id from context.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(requestIDKey{}).(string)
	return v, ok
}
