package revisor

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/Semior001/newsfeed/app/store"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/exp/slog"
)

//go:embed data/prompt.tmpl
var prompt string

var promptTmpl = template.Must(template.New("prompt").Parse(prompt))

//go:generate moq -out openai_client_mock.go . OpenAIClient
// OpenAIClient is interface for OpenAI client with the possibility to mock it
type OpenAIClient interface {
	CreateChatCompletion(context.Context, openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

// ChatGPT is a client to make requests to OpenAI chatgpt service.
type ChatGPT struct {
	log       *slog.Logger
	cl        OpenAIClient
	maxTokens int
}

// NewChatGPT creates new ChatGPT client.
func NewChatGPT(lg *slog.Logger, cl *http.Client, token string, maxTokens int) *ChatGPT {
	config := openai.DefaultConfig(token)
	config.HTTPClient = cl

	client := openai.NewClientWithConfig(config)

	return &ChatGPT{
		log:       lg,
		cl:        &loggingClient{log: lg, cl: client},
		maxTokens: maxTokens,
	}
}

// BulletPoints shortens article.
func (s *ChatGPT) BulletPoints(ctx context.Context, article store.Article) (string, error) {
	buf := &strings.Builder{}

	if err := promptTmpl.Execute(buf, article); err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: s.maxTokens,
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

	return resp.Choices[0].Message.Content, nil
}

type loggingClient struct {
	log *slog.Logger
	cl  OpenAIClient
}

func (l *loggingClient) CreateChatCompletion(
	ctx context.Context,
	req openai.ChatCompletionRequest,
) (openai.ChatCompletionResponse, error) {
	l.log.DebugCtx(ctx, "sending request to chatGPT")
	resp, err := l.cl.CreateChatCompletion(ctx, req)
	l.log.DebugCtx(ctx, "response received from chatGPT")
	return resp, err
}
