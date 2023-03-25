package bot

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"text/template"

	"github.com/Semior001/newsfeed/app/revisor"
	"github.com/Semior001/newsfeed/app/store"
	"github.com/Semior001/newsfeed/pkg/botx"
)

type article struct {
	API     botx.API
	Service *revisor.Service
}

var articleMessageTmpl = template.Must(template.New("articleMessage").Parse(`
*{{.Title}} by {{.Author}}*

{{.BulletPoints}}

[source]({{.URL}})
`))

func (c *article) article(ctx context.Context, req botx.Request) ([]botx.Response, error) {
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
