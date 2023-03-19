package revisor

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Semior001/newsfeed/app/store"
	"github.com/jessevdk/go-flags"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

//go:embed data/test/chatgpt_request.txt
var chatGPTRequest string

func TestChatGPT_Integration(t *testing.T) {
	var opts struct {
		Token string `env:"OPENAI_TOKEN"`
	}

	_, err := flags.NewParser(&opts, flags.Default|flags.IgnoreUnknown).Parse()
	require.NoError(t, err)

	cl := NewChatGPT(slog.Default(), &http.Client{}, opts.Token, 1000)

	resp, err := cl.cl.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: cl.maxTokens,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: "are you alive?"},
		},
	})
	require.NoError(t, err)

	t.Log(resp)
}

func TestChatGPT_Shorten(t *testing.T) {
	cl := &ChatGPT{
		log: slog.Default(),
		cl: &OpenAIClientMock{
			CreateChatCompletionFunc: func(
				ctx context.Context,
				req openai.ChatCompletionRequest,
			) (openai.ChatCompletionResponse, error) {
				assert.Equal(t, openai.ChatCompletionRequest{
					Model: openai.GPT3Dot5Turbo,
					Messages: []openai.ChatCompletionMessage{
						{Role: "user", Content: chatGPTRequest},
					},
					MaxTokens: 1000,
				}, req)
				return openai.ChatCompletionResponse{
					Choices: []openai.ChatCompletionChoice{{
						Message: openai.ChatCompletionMessage{
							Content: "shortened content",
						}},
					},
				}, nil
			},
		},
		maxTokens: 1000,
	}

	var article store.Article
	err := json.Unmarshal(articleContent, &article)
	require.NoError(t, err)

	resp, err := cl.BulletPoints(context.Background(), article)
	require.NoError(t, err)

	assert.Equal(t, "shortened content", resp)
}
