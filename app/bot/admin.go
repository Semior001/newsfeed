package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Semior001/newsfeed/app/revisor"
	"github.com/Semior001/newsfeed/app/store"
	"github.com/Semior001/newsfeed/pkg/botx"
)

type admin struct {
	Store   store.Interface
	Service *revisor.Service
}

func (c *admin) list(ctx context.Context, req botx.Request) ([]botx.Response, error) {
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

func (c *admin) delete(ctx context.Context, req botx.Request) ([]botx.Response, error) {
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

func (c *admin) cacheStats(_ context.Context, req botx.Request) ([]botx.Response, error) {
	stats := c.Service.GPTCacheStat()
	return []botx.Response{{
		ChatID: req.Chat.ID,
		Text: fmt.Sprintf("hits: %d, misses: %d, evictions: %d, size: %d\n",
			stats.Hits, stats.Misses, stats.Evicted, stats.Added),
	}}, nil
}
