package revisor

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/Semior001/newsfeed/app/store"
	"github.com/Semior001/newsfeed/pkg/logx"
	cache "github.com/go-pkgz/expirable-cache/v2"
	"github.com/go-pkgz/requester"
	"github.com/go-pkgz/requester/middleware"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/exp/slog"
)

//go:embed data/prompt.tmpl
var prompt string

var promptTmpl = template.Must(template.New("prompt").Parse(prompt))

//go:generate moq -out mock_openai_client.go . OpenAIClient

// OpenAIClient is interface for OpenAI client with the possibility to mock it
type OpenAIClient interface {
	CreateChatCompletion(context.Context, openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

// ChatGPT is a client to make requests to OpenAI chatgpt service.
type ChatGPT struct {
	log               *slog.Logger
	cl                OpenAIClient
	maxResponseTokens int
	cache             cache.Cache[string, string]
}

// NewChatGPT creates new ChatGPT client.
func NewChatGPT(lg *slog.Logger, cl http.Client, token string, maxResponseTokens int) *ChatGPT {
	rstr := requester.New(cl,
		middleware.MaxConcurrent(3),
		logx.LoggingRoundTripper(lg, logx.RoundTripperOpts{
			Level:         slog.LevelDebug,
			SecretHeaders: []string{"Authorization"},
		}),
	)

	config := openai.DefaultConfig(token)
	config.HTTPClient = rstr.Client()

	client := openai.NewClientWithConfig(config)

	svc := &ChatGPT{
		log:               lg,
		cl:                client,
		maxResponseTokens: maxResponseTokens,
		cache: cache.NewCache[string, string]().
			WithLRU().
			WithMaxKeys(100),
	}

	return svc
}

// maxRequestTokens is a maximum number of tokens that can be sent to OpenAI.
const maxRequestTokens = 4097

// ErrTooManyTokens is returned when article is too long.
var ErrTooManyTokens = fmt.Errorf("too many tokens")

// CacheStat returns cache stats.
func (s *ChatGPT) CacheStat() cache.Stats { return s.cache.Stat() }

// BulletPoints shortens article.
func (s *ChatGPT) BulletPoints(ctx context.Context, article store.Article) (string, error) {
	if resp, ok := s.cache.Get(article.URL); ok {
		return resp, nil
	}

	buf := &strings.Builder{}

	if err := promptTmpl.Execute(buf, article); err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	totalTokens := strings.Count(buf.String(), " ") + 1
	if totalTokens > maxRequestTokens {
		return "", ErrTooManyTokens
	}

	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: s.maxResponseTokens,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: buf.String()},
		},
	}

	resp, err := s.cl.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	result := resp.Choices[0].Message.Content
	s.cache.Set(article.URL, result, 0)
	return result, nil
}
