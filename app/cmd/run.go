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
	"github.com/Semior001/newsfeed/pkg/botx"
	"github.com/Semior001/newsfeed/pkg/botx/botapi"
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
			http.Client{Timeout: r.Revisor.OpenAI.Timeout},
			r.Revisor.OpenAI.Token,
			r.Revisor.OpenAI.MaxTokens,
		),
		revisor.NewExtractor(),
	)

	s, err := store.NewBolt(r.StorePath)
	if err != nil {
		return fmt.Errorf("make store: %w", err)
	}

	defer func() {
		if err := s.Close(); err != nil {
			lg.Error("close bolt store", slog.Any("err", err))
		}
	}()

	api, err := botapi.NewTelegram(
		lg.With(slog.String("prefix", "telegram")),
		r.Bot.Telegram.Token,
		100,
	)
	if err != nil {
		return fmt.Errorf("make telegram controller: %w", err)
	}

	ctrl := &bot.Ctrl{
		Logger:         lg.With(slog.String("prefix", "bot")),
		Store:          s,
		Service:        rev,
		API:            api,
		AdminIDs:       r.Bot.AdminIDs,
		AuthToken:      r.Bot.AuthToken,
		HandlerTimeout: r.Bot.Timeout,
	}

	b := botx.NewBot(
		ctrl.Routes().Handle,
		api,
		botx.WithLogger(lg.With(slog.String("prefix", "botx"))),
		botx.WithWorkers(10),
	)

	if err := ctrl.NotifyAdmins(context.Background(), "bot started"); err != nil {
		return fmt.Errorf("notify admins about started bot: %w", err)
	}

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
		lg.Info("starting bot")
		b.Run(ctx)
		lg.Warn("bot stopped")
		return nil
	})

	// we should run api out of errgroup, because it lives longer than the context,
	// as we want to notify admins about bot stopping
	apiStopped := make(chan struct{})
	go func() {
		lg.Info("starting telegram api")
		api.Run()
		lg.Warn("telegram api stopped listening for updates")
		apiStopped <- struct{}{}
	}()

	if err := ewg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		msg := fmt.Sprintf("bot stopped with error: %v", err)

		if sendErr := ctrl.NotifyAdmins(context.Background(), msg); sendErr != nil {
			return fmt.Errorf("notify admins about stopped bot (for reason: %v): %w", err, sendErr)
		}

		return err
	}

	if err := ctrl.NotifyAdmins(context.Background(), "bot stopped"); err != nil {
		return fmt.Errorf("notify admins about stopped bot: %w", err)
	}

	lg.Info("stopping telegram api")
	api.Stop()
	<-apiStopped
	lg.Info("telegram api stopped")

	return nil
}
