// Package revisor contains services for processing and retrieving articles.
package revisor

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Semior001/newsfeed/app/store"
	cache "github.com/go-pkgz/expirable-cache/v2"
	"golang.org/x/exp/slog"
)

// Service is a main application service.
type Service struct {
	log       *slog.Logger
	cl        *http.Client
	chatGPT   *ChatGPT
	extractor Extractor
}

// NewService creates new service.
func NewService(lg *slog.Logger, cl *http.Client, chatGPT *ChatGPT, extractor Extractor) *Service {
	return &Service{
		log:       lg,
		cl:        cl,
		chatGPT:   chatGPT,
		extractor: extractor,
	}
}

// GPTCacheStat returns cache stats.
func (s *Service) GPTCacheStat() cache.Stats { return s.chatGPT.CacheStat() }

// GetArticle shortens article.
func (s *Service) GetArticle(ctx context.Context, u string) (store.Article, error) {
	s.log.DebugCtx(ctx, "aggregating article from", slog.String("url", u))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return store.Article{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := s.cl.Do(req)
	if err != nil {
		return store.Article{}, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.log.WarnCtx(ctx, "failed to close response body", slog.Any("err", err))
		}
	}()

	ok := resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
	if !ok {
		return store.Article{}, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	article, err := s.extractor.Extract(resp.Body)
	if err != nil {
		return store.Article{}, fmt.Errorf("extract article: %w", err)
	}
	article.URL = u

	if article.BulletPoints, err = s.chatGPT.BulletPoints(ctx, article); err != nil {
		return store.Article{}, fmt.Errorf("get bullet points: %w", err)
	}

	return article, nil
}
