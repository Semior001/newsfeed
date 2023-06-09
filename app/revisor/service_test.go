package revisor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Semior001/newsfeed/app/store"
	cache "github.com/go-pkgz/expirable-cache/v2"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

func TestService_GetArticle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(articleHTML)
		require.NoError(t, err)
	}))
	defer ts.Close()

	svc := Service{
		log: slog.Default(),
		cl:  ts.Client(),
		chatGPT: &ChatGPT{
			cache: cache.NewCache[string, string](),
			log:   slog.Default(),
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
		},
		extractor: Extractor{},
	}

	article, err := svc.GetArticle(context.Background(), ts.URL)
	require.NoError(t, err)

	var expected store.Article
	err = json.Unmarshal(articleContent, &expected)
	require.NoError(t, err)
	expected.BulletPoints = "shortened content"
	expected.URL = ts.URL

	assert.Equal(t, expected, article)
}
