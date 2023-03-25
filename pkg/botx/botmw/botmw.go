// Package botmw provides middlewares for bot handler.
package botmw

import (
	"context"

	"github.com/Semior001/newsfeed/pkg/botx"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

// Logger is a middleware that logs all requests
func Logger(lg *slog.Logger) botx.Middleware {
	return func(next botx.Handler) botx.Handler {
		return func(ctx context.Context, req botx.Request) ([]botx.Response, error) {
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

			if lg.Handler().Enabled(ctx, slog.LevelDebug) {
				lg.DebugCtx(ctx, "request processed", slog.Any("responses", res), slog.Any("err", err))
				return res, err
			}

			lg.InfoCtx(ctx, "request processed",
				slog.Any("responses", lo.Map(res, func(r botx.Response, _ int) botx.Response {
					return botx.Response{ChatID: r.ChatID}
				})),
				slog.Any("err", err),
			)

			return res, err
		}
	}
}

// Recover is a middleware that recovers from panics.
func Recover(lg *slog.Logger) botx.Middleware {
	return func(next botx.Handler) botx.Handler {
		return func(ctx context.Context, req botx.Request) ([]botx.Response, error) {
			defer func() {
				if r := recover(); r != nil {
					lg.ErrorCtx(ctx, "panic recovered", slog.Any("panic", r))
				}
			}()

			return next(ctx, req)
		}
	}
}
