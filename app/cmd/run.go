// Package cmd contains commands for the application.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Semior001/newsfeed/app/bot"
	"github.com/Semior001/newsfeed/app/revisor"
	"github.com/Semior001/newsfeed/app/store"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
)

// Run is a command to run the bot.
type Run struct {
	Timeout  time.Duration `long:"timeout" env:"TIMEOUT" default:"5s" description:"timeout for http calls to articles"`
	Telegram struct {
		Token string `long:"token" env:"TOKEN" description:"telegram token"`
	} `group:"telegram" namespace:"telegram" env-namespace:"TELEGRAM"`
	OpenAI struct {
		Token     string        `long:"token" env:"TOKEN" description:"OpenAI token"`
		MaxTokens int           `long:"max-tokens" env:"MAX_TOKENS" default:"1000" description:"max tokens for OpenAI"`
		Timeout   time.Duration `long:"timeout" env:"TIMEOUT" default:"5m" description:"timeout for OpenAI calls"`
	} `group:"openai" namespace:"openai" env-namespace:"OPENAI"`
	AdminIDs  []string `long:"admin-ids" env:"ADMIN_IDS" description:"admin IDs"`
	AuthToken string   `long:"auth-token" env:"AUTH_TOKEN" description:"token for authorizing requests"`
	StorePath string   `long:"store-path" env:"STORE_PATH" description:"parent dir for bolt files"`
}

// Execute runs the command.
func (r Run) Execute(_ []string) error {
	lg := slog.Default()

	ctrl, err := bot.NewTelegram(lg.WithGroup("telegram"), r.Telegram.Token)
	if err != nil {
		return fmt.Errorf("make telegram controller: %w", err)
	}

	s, err := store.NewBolt(r.StorePath)
	if err != nil {
		return fmt.Errorf("make store: %w", err)
	}

	httpCl := &http.Client{Timeout: r.Timeout}

	chatGPT := revisor.NewChatGPT(
		lg.WithGroup("chatgpt"),
		&http.Client{Timeout: r.OpenAI.Timeout},
		r.OpenAI.Token,
		r.OpenAI.MaxTokens,
	)

	extractor := revisor.NewExtractor()

	svc := revisor.NewService(
		lg.WithGroup("revisor"),
		httpCl,
		chatGPT,
		extractor,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	b := bot.New(lg, ctrl, s, svc, bot.Params{
		AdminIDs:  r.AdminIDs,
		AuthToken: r.AuthToken,
		Workers:   10,
	})

	ewg, ctx := errgroup.WithContext(ctx)
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

	if err := ewg.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}

		return err
	}

	return nil
}
