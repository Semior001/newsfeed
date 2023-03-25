// Package botapi contains implementations of bot API interfaces.
package botapi

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Semior001/newsfeed/pkg/botx"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/slog"
)

// Telegram is a controller that handles requests from telegram.
type Telegram struct {
	api     *tgbotapi.BotAPI
	updates chan botx.Request
}

// NewTelegram returns a new telegram bot controller.
func NewTelegram(lg *slog.Logger, token string, bufferSize int) (*Telegram, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("make new api: %w", err)
	}

	stdlibLogger := slog.NewLogLogger(lg.Handler(), slog.LevelWarn)
	stdlibLogger.SetPrefix("telegram-bot-api: ")

	if err = tgbotapi.SetLogger(stdlibLogger); err != nil {
		return nil, fmt.Errorf("set logger: %w", err)
	}

	return &Telegram{
		api:     api,
		updates: make(chan botx.Request, bufferSize),
	}, nil
}

// Run runs telegram bot listener until context is dead.
func (b *Telegram) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for {
		update, ok := <-updates
		if !ok {
			return
		}

		if update.Message == nil || update.Message.Chat == nil || update.Message.Text == "" {
			continue
		}

		b.updates <- botx.Request{
			Chat: botx.Chat{
				ID:       strconv.FormatInt(update.Message.Chat.ID, 10),
				Username: update.Message.Chat.UserName,
			},
			Text: update.Message.Text,
		}
	}
}

// Stop stops telegram bot listener.
func (b *Telegram) Stop() {
	b.api.StopReceivingUpdates()
	close(b.updates)
}

// Updates returns updates channel.
func (b *Telegram) Updates() <-chan botx.Request {
	return b.updates
}

// SendMessage sends message to telegram user.
func (b *Telegram) SendMessage(ctx context.Context, resp botx.Response) error {
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
	if resp.ReplyToMessageID != "" {
		if msg.ReplyToMessageID, err = strconv.Atoi(resp.ReplyToMessageID); err != nil {
			return fmt.Errorf("parse reply to message id: %w", err)
		}
	}

	if _, err = b.api.Send(msg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}
