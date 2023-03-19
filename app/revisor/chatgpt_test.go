package revisor

import (
	"context"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/Semior001/newsfeed/app/store"
	cache "github.com/go-pkgz/expirable-cache/v2"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

//go:embed data/test/chatgpt_request.txt
var chatGPTRequest string

func TestChatGPT_Shorten(t *testing.T) {
	cl := &ChatGPT{
		log:   slog.Default(),
		cache: cache.NewCache[string, string](),
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
		maxResponseTokens: 1000,
	}

	var article store.Article
	err := json.Unmarshal(articleContent, &article)
	require.NoError(t, err)

	resp, err := cl.BulletPoints(context.Background(), article)
	require.NoError(t, err)

	assert.Equal(t, "shortened content", resp)
}
