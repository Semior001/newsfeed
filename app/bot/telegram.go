package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Semior001/newsfeed/app/bot/route"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/slog"
)

// Telegram is a controller that handles requests from telegram.
type Telegram struct {
	log     *slog.Logger
	api     *tgbotapi.BotAPI
	updates chan route.Request
}

// NewTelegram returns a new telegram bot controller.
func NewTelegram(lg *slog.Logger, token string) (*Telegram, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("make new api: %w", err)
	}

	api.Debug = lg.Enabled(context.TODO(), slog.LevelDebug)

	stdlibLogger := slog.NewLogLogger(lg.Handler(), slog.LevelWarn)
	stdlibLogger.SetPrefix("telegram-bot-api: ")

	if err = tgbotapi.SetLogger(stdlibLogger); err != nil {
		return nil, fmt.Errorf("set logger: %w", err)
	}

	return &Telegram{
		log:     lg,
		api:     api,
		updates: make(chan route.Request, 100),
	}, nil
}

// Run starts telegram bot listener.
func (b *Telegram) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	b.log.InfoCtx(ctx, "started bot listener")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case update, ok := <-updates:
			if !ok {
				return errors.New("telegram updates chan closed")
			}
			if update.Message == nil || update.Message.Chat == nil || update.Message.Text == "" {
				continue
			}

			b.updates <- route.Request{
				Chat: route.Chat{
					ID:       strconv.FormatInt(update.Message.Chat.ID, 10),
					Username: update.Message.Chat.UserName,
				},
				Text: update.Message.Text,
			}
		}
	}
}

// Updates returns updates channel.
func (b *Telegram) Updates() <-chan route.Request {
	return b.updates
}

// SendMessage sends message to telegram user.
func (b *Telegram) SendMessage(ctx context.Context, resp route.Response) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	chatID, err := strconv.ParseInt(resp.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("parse chat id: %w", err)
	}

	msg := tgbotapi.NewMessage(chatID, resp.Text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.DisableWebPagePreview = true

	if _, err = b.api.Send(msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}
