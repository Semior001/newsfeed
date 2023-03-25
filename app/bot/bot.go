// Package bot contains routers and controllers for bots.
package bot

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"text/template"
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

	rtr.NotFound(c.article)
	rtr.Add("/start", c.start)
	rtr.Add("/stop", c.stop)

	rtr.Group(func(rtr *botx.Router) {
		rtr.Use(c.ensureAdmin)

		rtr.Add("/list", c.list)
		rtr.Add("/delete", c.delete)
		rtr.Add("/cache", c.cacheStats)
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

func (c *Ctrl) list(ctx context.Context, req botx.Request) ([]botx.Response, error) {
	users, err := c.Store.List(ctx, store.ListRequest{})
	if err != nil {
		return nil, fmt.Errorf("list subscribers: %w", err)
	}

	sb := &strings.Builder{}
	_, _ = sb.WriteString("Subscribers:\n")
	for _, u := range users {
		_, _ = sb.WriteString(fmt.Sprintf("id: %s, username: %s, authorized: %t, subscribed: %t\n",
			u.ChatID, escapeMarkdown(u.Username), u.Authorized, u.Subscribed))
	}

	return []botx.Response{{
		ChatID: req.Chat.ID,
		Text:   sb.String(),
	}}, nil
}

func (c *Ctrl) delete(ctx context.Context, req botx.Request) ([]botx.Response, error) {
	tokens := strings.Split(req.Text, " ")
	if len(tokens) != 2 {
		return nil, errors.New("invalid command")
	}

	chatID := tokens[1]
	if err := c.Store.Delete(ctx, chatID); err != nil {
		return nil, fmt.Errorf("delete user: %w", err)
	}

	return []botx.Response{{
		ChatID: req.Chat.ID,
		Text:   fmt.Sprintf("User with id %s was deleted.", chatID),
	}}, nil
}

func (c *Ctrl) cacheStats(_ context.Context, req botx.Request) ([]botx.Response, error) {
	stats := c.Service.GPTCacheStat()
	return []botx.Response{{
		ChatID: req.Chat.ID,
		Text: fmt.Sprintf("hits: %d, misses: %d, evictions: %d, size: %d\n",
			stats.Hits, stats.Misses, stats.Evicted, stats.Added),
	}}, nil
}

var articleMessageTmpl = template.Must(template.New("articleMessage").Parse(`
*{{.Title}} by {{.Author}}*

{{.BulletPoints}}

[source]({{.URL}})
`))

func (c *Ctrl) article(ctx context.Context, req botx.Request) ([]botx.Response, error) {
	if _, err := url.ParseRequestURI(req.Text); err != nil {
		return []botx.Response{{
			ChatID: req.Chat.ID,
			Text: "Please, send me just a link without any other text.\n" +
				"You can send me a link to any article, in order to test my capability of shortening it.\n" +
				"But do not overuse it, please, we don't have an unlimited amount of free API calls.",
		}}, nil
	}

	err := c.API.SendMessage(ctx, botx.Response{
		ChatID: req.Chat.ID,
		Text:   "I'm working on it, please wait...",
	})
	if err != nil {
		return nil, fmt.Errorf("send start message: %w", err)
	}

	article, err := c.Service.GetArticle(ctx, req.Text)
	if err != nil {
		if errors.Is(err, revisor.ErrTooManyTokens) {
			return []botx.Response{{
				ChatID: req.Chat.ID,
				Text: "Article you provided is too long, I can't summarize it.\n" +
					"Article content should be less than 4000 words.",
			}}, nil
		}
		return nil, fmt.Errorf("get article: %w", err)
	}

	sb := &strings.Builder{}
	if err = articleMessageTmpl.Execute(sb, escapeArticle(article)); err != nil {
		return nil, fmt.Errorf("execute article message template: %w", err)
	}

	return []botx.Response{{
		ChatID: req.Chat.ID,
		Text:   sb.String(),
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

func escapeArticle(a store.Article) store.Article {
	a.Title = escapeMarkdown(a.Title)
	a.Author = escapeMarkdown(a.Author)
	a.Excerpt = escapeMarkdown(a.Excerpt)
	return a
}

func escapeMarkdown(s string) string {
	return mdEscaper.Replace(s)
}
