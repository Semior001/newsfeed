// Package bot contains routers and controllers for bots.
package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Semior001/newsfeed/app/revisor"
	"github.com/Semior001/newsfeed/app/store"
	"github.com/Semior001/newsfeed/pkg/botx"
	"github.com/Semior001/newsfeed/pkg/botx/botmw"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
)

// Ctrl provides routes and controllers for bot updates.
type Ctrl struct {
	Logger         *slog.Logger
	Store          store.Interface
	Service        *revisor.Service
	API            botx.API
	AdminIDs       []string
	AuthToken      string
	HandlerTimeout time.Duration
}

// Routes returns a multiplexer for bot controllers.
func (c *Ctrl) Routes() *botx.Router {
	rtr := botx.NewRouter()

	rtr.Use(
		botmw.RequestID(),
		botmw.AppendRequestIDOnError(),
		botmw.Recover(c.Logger),
		botmw.Logger(c.Logger),
		botmw.Timeout(c.HandlerTimeout),
		c.ensureAuthorized,
	)

	articleCtrl := &article{
		API:     c.API,
		Service: c.Service,
	}
	rtr.NotFound(articleCtrl.article)

	rtr.Add("/start", c.start)
	rtr.Add("/stop", c.stop)

	rtr.Group(func(rtr *botx.Router) {
		rtr.Use(c.ensureAdmin)

		adminCtrl := &admin{
			Store:   c.Store,
			Service: c.Service,
		}

		rtr.Add("/list", adminCtrl.list)
		rtr.Add("/delete", adminCtrl.delete)
		rtr.Add("/cache", adminCtrl.cacheStats)
	})

	return rtr
}

func (c *Ctrl) start(ctx context.Context, req botx.Request) ([]botx.Response, error) {
	u, ok := userFromContext(ctx)
	if !ok {
		return c.register(ctx, req)
	}

	u.Subscribed = true
	if err := c.Store.Put(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return []botx.Response{{
		ChatID: req.Chat.ID,
		Text:   "You have been subscribed to news updates.",
	}}, nil
}

func (c *Ctrl) stop(ctx context.Context, req botx.Request) ([]botx.Response, error) {
	u, ok := userFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no user in context")
	}

	u.Subscribed = false
	if err := c.Store.Put(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return []botx.Response{{
		ChatID: req.Chat.ID,
		Text:   "You will no longer receive news updates.",
	}}, nil
}

func (c *Ctrl) ensureAdmin(h botx.Handler) botx.Handler {
	return func(ctx context.Context, req botx.Request) ([]botx.Response, error) {
		if !lo.Contains(c.AdminIDs, req.Chat.ID) {
			return nil, nil
		}

		return h(ctx, req)
	}
}

func (c *Ctrl) ensureAuthorized(h botx.Handler) botx.Handler {
	return func(ctx context.Context, req botx.Request) ([]botx.Response, error) {
		u, err := c.Store.Get(ctx, req.Chat.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return c.register(ctx, req)
			}

			return nil, fmt.Errorf("get user: %w", err)
		}

		if !u.Authorized {
			if req.Text != c.AuthToken {
				return []botx.Response{{
					ChatID: req.Chat.ID,
					Text:   "You are not authorized, please provide a token.",
				}}, nil
			}

			u.Authorized = true
			u.Subscribed = true

			if err := c.Store.Put(ctx, u); err != nil {
				return nil, fmt.Errorf("update user: %w", err)
			}

			return []botx.Response{{
				ChatID: req.Chat.ID,
				Text: "You are now authorized.\n" +
					"Now, you can send me a link to any article, in order to test my capability of shortening it.\n" +
					"But do not overuse it, please, we don't have an unlimited amount of free API calls.",
			}}, nil
		}

		return h(contextWithUser(ctx, u), req)
	}
}

func (c *Ctrl) register(ctx context.Context, req botx.Request) ([]botx.Response, error) {
	u := store.User{
		ChatID:   req.Chat.ID,
		Username: req.Chat.Username,
	}

	if err := c.Store.Put(ctx, u); err != nil {
		return nil, fmt.Errorf("add subscriber: %w", err)
	}

	const response = "Hello! In order to subscribe to news, you need to provide a token,\n" +
		"please ask admin for it and then send it to me."

	if err := c.NotifyAdmins(ctx, fmt.Sprintf("new user: %s", req.Chat.Username)); err != nil {
		c.Logger.WarnCtx(ctx, "notify admins about registered user", slog.Any("err", err))
	}

	return []botx.Response{{
		ChatID: req.Chat.ID,
		Text:   response,
	}}, nil
}

// NotifyAdmins sends a message to all admins.
func (c *Ctrl) NotifyAdmins(ctx context.Context, msg string) error {
	for _, adminID := range c.AdminIDs {
		if err := c.API.SendMessage(ctx, botx.Response{
			ChatID: adminID,
			Text:   msg,
		}); err != nil {
			return fmt.Errorf("send message to admin: %w", err)
		}
	}

	return nil
}

var mdEscaper = strings.NewReplacer(
	`*`, `\*`,
	`_`, `\_`,
	"`", "\\`",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	"~", "\\~",
	">", "\\>",
)

func escapeMarkdown(s string) string {
	return mdEscaper.Replace(s)
}
