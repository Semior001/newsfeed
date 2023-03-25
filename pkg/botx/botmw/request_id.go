package botmw

import (
	"context"
	"fmt"

	"github.com/Semior001/newsfeed/pkg/botx"
	"github.com/Semior001/newsfeed/pkg/logx"
	"github.com/google/uuid"
)

// RequestID is a middleware that adds request id to context.
func RequestID() botx.Middleware {
	return func(next botx.Handler) botx.Handler {
		return func(ctx context.Context, req botx.Request) ([]botx.Response, error) {
			id := uuid.New().String()
			ctx = logx.ContextWithRequestID(ctx, id)

			return next(ctx, req)
		}
	}
}

// AppendRequestIDOnError is a middleware that responds with error message.
func AppendRequestIDOnError() botx.Middleware {
	return func(next botx.Handler) botx.Handler {
		return func(ctx context.Context, req botx.Request) (resps []botx.Response, err error) {
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
				resps = append(resps, botx.Response{
					ChatID: req.Chat.ID,
					Text: fmt.Sprintf("Something went wrong. "+
						"Please, ask admin for help."+
						"\n\nRequest ID: `%s`", reqID),
				})
			}

			return resps, err
		}
	}
}
