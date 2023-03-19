// Package cmd contains commands for the application.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Semior001/newsfeed/app/bot"
	"github.com/Semior001/newsfeed/app/revisor"
	"github.com/Semior001/newsfeed/app/store"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
)

// Run is a command to run the bot.
type Run struct {
	Bot struct {
		Timeout time.Duration `long:"timeout" env:"TIMEOUT" default:"6m" description:"timeout for requests"`

		Telegram struct {
			Token string `long:"token" env:"TOKEN" description:"telegram token"`
		} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`

		AdminIDs  []string `long:"admin-ids" env:"ADMIN_IDS" description:"admin IDs"`
		AuthToken string   `long:"auth-token" env:"AUTH_TOKEN" description:"token for authorizing requests"`
	} `group:"bot" namespace:"bot" env-namespace:"BOT"`

	Revisor struct {
		OpenAI struct {
			Token     string        `long:"token" env:"TOKEN" description:"OpenAI token"`
			MaxTokens int           `long:"max-tokens" env:"MAX_TOKENS" default:"1000" description:"max tokens for OpenAI"`
			Timeout   time.Duration `long:"timeout" env:"TIMEOUT" default:"5m" description:"timeout for OpenAI calls"`
		} `group:"openai" namespace:"openai" env-namespace:"OPENAI"`
	} `group:"revisor" namespace:"revisor" env-namespace:"REVISOR"`

	StorePath string `long:"store-path" env:"STORE_PATH" description:"parent dir for bolt files"`
}

// Execute runs the command.
func (r Run) Execute(_ []string) error {
	lg := slog.Default()

	rev := revisor.NewService(
		lg.With(slog.String("prefix", "revisor")),
		&http.Client{Timeout: 5 * time.Second},
		revisor.NewChatGPT(
			lg.With(slog.String("prefix", "chatgpt")),
			&http.Client{Timeout: r.Revisor.OpenAI.Timeout},
			r.Revisor.OpenAI.Token,
			r.Revisor.OpenAI.MaxTokens,
		),
		revisor.NewExtractor(),
	)

	s, err := store.NewBolt(r.StorePath)
	if err != nil {
		return fmt.Errorf("make store: %w", err)
	}

	ctrl, err := bot.NewTelegram(lg.With(slog.String("prefix", "telegram")), r.Bot.Telegram.Token)
	if err != nil {
		return fmt.Errorf("make telegram controller: %w", err)
	}

	b := bot.New(lg, ctrl, s, rev, bot.Params{
		AdminIDs:  r.Bot.AdminIDs,
		AuthToken: r.Bot.AuthToken,
		Timeout:   r.Bot.Timeout,
		Workers:   10,
	})

	ctx, stop := context.WithCancel(context.Background())

	ewg, ctx := errgroup.WithContext(ctx)
	ewg.Go(func() error {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		select {
		case sig := <-sig:
			slog.Warn("caught signal, stopping", slog.String("signal", sig.String()))
			stop()
			return ctx.Err()
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	ewg.Go(func() error {
		if err := b.Run(ctx); err != nil {
			return fmt.Errorf("bot stopped, reason: %w", err)
		}
		return nil
	})
	ewg.Go(func() error {
		if err := ctrl.Run(ctx); err != nil {
			return fmt.Errorf("controller stopped, reason: %w", err)
		}
		return nil
	})

	if err := ewg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
