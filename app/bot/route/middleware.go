package route

import (
	"context"
	"fmt"

	"github.com/Semior001/newsfeed/pkg/logx"
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

			if lg.Handler().Enabled(ctx, slog.LevelDebug) {
				lg.DebugCtx(ctx, "request processed", slog.Any("responses", res), slog.Any("err", err))
				return res, err
			}

			lg.InfoCtx(ctx, "request processed",
				slog.Any("responses", lo.Map(res, func(r Response, _ int) Response {
					return Response{ChatID: r.ChatID}
				})),
				slog.Any("err", err),
			)

			return res, err
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
func RequestID(next Handler) Handler {
	return func(ctx context.Context, req Request) ([]Response, error) {
		id := uuid.New().String()
		ctx = logx.ContextWithRequestID(ctx, id)

		return next(ctx, req)
	}
}

// AppendRequestIDOnError is a middleware that responds with error message.
func AppendRequestIDOnError(next Handler) Handler {
	return func(ctx context.Context, req Request) (resps []Response, err error) {
		resps, err = next(ctx, req)
		if err == nil {
			return resps, nil
		}

		reqID, _ := logx.RequestIDFromContext(ctx)

		hasRequester := false
		for i := range resps {
			resps[i].Text += fmt.Sprintf("\n\nRequest ID: `%s`", reqID)
			if resps[i].ChatID == req.Chat.ID {
				hasRequester = true
			}
		}

		if !hasRequester {
			resps = append(resps, Response{
				ChatID: req.Chat.ID,
				Text: fmt.Sprintf("Something went wrong. "+
					"Please, ask admin for help."+
					"\n\nRequest ID: `%s`", reqID),
			})
		}

		return resps, err
	}
}
