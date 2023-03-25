// Package botx provides interfaces and types to handle bot updates,
// with a chi-like router.
package botx

import (
	"context"
	"sync"

	"github.com/Semior001/newsfeed/pkg/logx"
	"golang.org/x/exp/slog"
)

// API defines methods for an API interface to receive and send chat messages.
type API interface {
	Updates() <-chan Request
	SendMessage(ctx context.Context, resp Response) error
}

// Bot defines parameters for running a bot over some API.
type Bot struct {
	h   Handler
	api API
	Options
}

// NewBot creates a new Bot.
func NewBot(h Handler, api API, opts ...Option) *Bot {
	options := Options{
		Workers: 1,
		Logger:  slog.New(logx.NoOp()),
	}

	for _, opt := range opts {
		opt(&options)
	}

	return &Bot{
		h:       h,
		api:     api,
		Options: options,
	}
}

// Run starts updates listener.
func (b *Bot) Run(ctx context.Context) {
	wg := &sync.WaitGroup{}
	wg.Add(b.Workers)

	for i := 0; i < b.Workers; i++ {
		go func(idx int) {
			b.Logger.InfoCtx(ctx, "starting worker", slog.Int("worker", idx))

			defer func() {
				b.Logger.InfoCtx(ctx, "stopping worker", slog.Int("worker", idx))
				wg.Done()
			}()

			for {
				select {
				case <-ctx.Done():
					return
				case req := <-b.api.Updates():
					b.handleUpdate(ctx, req)
				}
			}
		}(i)
	}

	wg.Wait()
}

func (b *Bot) handleUpdate(ctx context.Context, req Request) {
	resps, err := b.h(ctx, req)
	if err != nil {
		b.Logger.ErrorCtx(ctx, "failed to handle request", slog.Any("err", err))
	}

	for _, resp := range resps {
		if err := b.api.SendMessage(ctx, resp); err != nil {
			b.Logger.WarnCtx(ctx, "failed to send message", slog.Any("err", err))
		}
	}
}
