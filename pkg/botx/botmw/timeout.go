package botmw

import (
	"context"
	"errors"
	"time"

	"github.com/Semior001/newsfeed/pkg/botx"
)

// ErrTimeout is returned by Timeout middleware when handler timed out.
var ErrTimeout = errors.New("timed out")

// Timeout sets the timeout for handler.
func Timeout(dur time.Duration) botx.Middleware {
	return func(next botx.Handler) botx.Handler {
		return func(ctx context.Context, req botx.Request) (resp []botx.Response, err error) {
			// set context timeout additionally
			ctx, cancel := context.WithTimeout(ctx, dur)
			defer cancel()

			done := make(chan struct{})

			go func() {
				resp, err = next(ctx, req)
				done <- struct{}{}
			}()

			timer := time.NewTimer(dur)
			defer timer.Stop()

			select {
			case <-done:
				return resp, err
			case <-timer.C:
				return nil, ErrTimeout
			}
		}
	}
}
