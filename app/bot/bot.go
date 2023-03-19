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

	"github.com/Semior001/newsfeed/app/bot/route"
	"github.com/Semior001/newsfeed/app/revisor"
	"github.com/Semior001/newsfeed/app/store"
	"github.com/samber/lo"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
)

// Controller defines methods for controller.
type Controller interface {
	Updates() <-chan route.Request
	SendMessage(ctx context.Context, resp route.Response) error
}

// Bot defines methods for service.
type Bot struct {
	logger *slog.Logger
	ctrl   Controller
	store  store.Interface
	svc    *revisor.Service

	h route.Handler

	Params
}

// Params defines service parameters.
type Params struct {
	AdminIDs  []string
	AuthToken string
	Workers   int
	Timeout   time.Duration
}

// New creates new service.
func New(lg *slog.Logger, ctrl Controller, s store.Interface, svc *revisor.Service, params Params) *Bot {
	bot := &Bot{
		logger: lg,
		ctrl:   ctrl,
		svc:    svc,
		store:  s,
		Params: params,
	}

	rtr := route.Router(map[string]route.Handler{
		"": bot.ensureAuthorized(route.Router(map[string]route.Handler{
			"":        bot.article,
			"/start":  bot.start,
			"/stop":   bot.stop,
			"/list":   bot.ensureAdmin(bot.list),
			"/delete": bot.ensureAdmin(bot.delete),
			"/cache":  bot.ensureAdmin(bot.cacheStats),
		})),
	})

	rtr = route.RequestID()(
		route.AppendRequestIDOnError()(
			route.Recover(lg)(
				route.Logger(lg)(rtr),
			),
		),
	)

	bot.h = rtr

	return bot
}

// Run starts service until context is dead.
func (b *Bot) Run(ctx context.Context) error {
	for _, id := range b.AdminIDs {
		msg := route.Response{ChatID: id, Text: "Bot started"}
		if err := b.ctrl.SendMessage(ctx, msg); err != nil {
			return fmt.Errorf("send start message to admins: %w", err)
		}
	}

	ewg, ctx := errgroup.WithContext(ctx)
	for i := 0; i < b.Workers; i++ {
		ewg.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case req := <-b.ctrl.Updates():
					b.handleUpdate(ctx, req)
				}
			}
		})
	}

	if err := ewg.Wait(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	for _, id := range b.AdminIDs {
		msg := route.Response{ChatID: id, Text: "Bot stopped"}
		if err := b.ctrl.SendMessage(ctx, msg); err != nil {
			return fmt.Errorf("send start message to admins: %w", err)
		}
	}

	return nil
}

func (b *Bot) handleUpdate(ctx context.Context, req route.Request) {
	if b.Timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, b.Timeout)
		defer cancel()
	}

	resps, err := b.h(ctx, req)
	if err != nil {
		b.logger.WarnCtx(ctx, "failed to handle request", slog.Any("err", err))
	}

	for _, resp := range resps {
		if err := b.ctrl.SendMessage(ctx, resp); err != nil {
			b.logger.WarnCtx(ctx, "failed to send message", slog.Any("err", err))
		}
	}
}

func (b *Bot) start(ctx context.Context, req route.Request) ([]route.Response, error) {
	u, ok := userFromContext(ctx)
	if !ok {
		return b.register(ctx, req)
	}

	u.Subscribed = true
	if err := b.store.Put(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return []route.Response{{
		ChatID: req.Chat.ID,
		Text:   "You have been subscribed to news updates.",
	}}, nil
}

func (b *Bot) stop(ctx context.Context, req route.Request) ([]route.Response, error) {
	u, ok := userFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no user in context")
	}

	u.Subscribed = false
	if err := b.store.Put(ctx, u); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return []route.Response{{
		ChatID: req.Chat.ID,
		Text:   "You will no longer receive news updates.",
	}}, nil
}

func (b *Bot) list(ctx context.Context, req route.Request) ([]route.Response, error) {
	users, err := b.store.List(ctx, store.ListRequest{})
	if err != nil {
		return nil, fmt.Errorf("list subscribers: %w", err)
	}

	sb := &strings.Builder{}
	_, _ = sb.WriteString("Subscribers:\n")
	for _, u := range users {
		_, _ = sb.WriteString(fmt.Sprintf("id: %s, username: %s, authorized: %t, subscribed: %t\n",
			u.ChatID, u.Username, u.Authorized, u.Subscribed))
	}

	return []route.Response{{
		ChatID: req.Chat.ID,
		Text:   sb.String(),
	}}, nil
}

func (b *Bot) delete(ctx context.Context, req route.Request) ([]route.Response, error) {
	tokens := strings.Split(req.Text, " ")
	if len(tokens) != 2 {
		return nil, errors.New("invalid command")
	}

	chatID := tokens[1]
	if err := b.store.Delete(ctx, chatID); err != nil {
		return nil, fmt.Errorf("delete user: %w", err)
	}

	return []route.Response{{
		ChatID: req.Chat.ID,
		Text:   fmt.Sprintf("User with id %s was deleted.", chatID),
	}}, nil
}

func (b *Bot) cacheStats(_ context.Context, req route.Request) ([]route.Response, error) {
	stats := b.svc.GPTCacheStat()
	return []route.Response{{
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

func (b *Bot) article(ctx context.Context, req route.Request) ([]route.Response, error) {
	if _, err := url.ParseRequestURI(req.Text); err != nil {
		return []route.Response{{
			ChatID: req.Chat.ID,
			Text: "I couldn't find any links in your message.\n" +
				"You can send me a link to any article, in order to test my capability of shortening it.\n" +
				"But do not overuse it, please, we don't have an unlimited amount of free API calls.",
		}}, nil
	}

	err := b.ctrl.SendMessage(ctx, route.Response{
		ChatID: req.Chat.ID,
		Text:   "I'm working on it, please wait...",
	})
	if err != nil {
		return nil, fmt.Errorf("send start message: %w", err)
	}

	article, err := b.svc.GetArticle(ctx, req.Text)
	if err != nil {
		if errors.Is(err, revisor.ErrTooManyTokens) {
			return []route.Response{{
				ChatID: req.Chat.ID,
				Text: "Article you provided is too long, I can't summarize it.\n" +
					"Article content should be less than 4000 words.",
			}}, nil
		}
		return nil, fmt.Errorf("get article: %w", err)
	}

	sb := &strings.Builder{}
	if err = articleMessageTmpl.Execute(sb, article); err != nil {
		return nil, fmt.Errorf("execute article message template: %w", err)
	}

	return []route.Response{{
		ChatID: req.Chat.ID,
		Text:   sb.String(),
	}}, nil
}

func (b *Bot) ensureAdmin(h route.Handler) route.Handler {
	return func(ctx context.Context, req route.Request) ([]route.Response, error) {
		if !lo.Contains(b.AdminIDs, req.Chat.ID) {
			return nil, nil
		}

		return h(ctx, req)
	}
}

func (b *Bot) ensureAuthorized(h route.Handler) route.Handler {
	return func(ctx context.Context, req route.Request) ([]route.Response, error) {
		u, err := b.store.Get(ctx, req.Chat.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return b.register(ctx, req)
			}

			return nil, fmt.Errorf("get user: %w", err)
		}

		if !u.Authorized {
			if req.Text == b.AuthToken {
				u.Authorized = true
				u.Subscribed = true

				if err := b.store.Put(ctx, u); err != nil {
					return nil, fmt.Errorf("update user: %w", err)
				}

				return []route.Response{{
					ChatID: req.Chat.ID,
					Text: "You are now authorized.\n" +
						"Now, you can send me a link to any article, in order to test my capability of shortening it.\n" +
						"But do not overuse it, please, we don't have an unlimited amount of free API calls.",
				}}, nil
			}

			return []route.Response{{
				ChatID: req.Chat.ID,
				Text:   "You are not authorized, please provide a token.",
			}}, nil
		}

		return h(contextWithUser(ctx, u), req)
	}
}

func (b *Bot) register(ctx context.Context, req route.Request) ([]route.Response, error) {
	u := store.User{
		ChatID:   req.Chat.ID,
		Username: req.Chat.Username,
	}

	if err := b.store.Put(ctx, u); err != nil {
		return nil, fmt.Errorf("add subscriber: %w", err)
	}

	const response = "Hello! In order to subscribe to news, you need to provide a token,\n" +
		"please ask admin for it and then send it to me."

	return []route.Response{{
		ChatID: req.Chat.ID,
		Text:   response,
	}}, nil
}
