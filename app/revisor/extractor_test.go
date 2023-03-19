package revisor

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/Semior001/newsfeed/app/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed data/test/article.html
var articleHTML []byte

//go:embed data/test/article.json
var articleContent []byte

func TestExtractor_Extract(t *testing.T) {
	article, err := Extractor{}.Extract(bytes.NewReader(articleHTML))
	require.NoError(t, err)

	var expected store.Article
	err = json.Unmarshal(articleContent, &expected)
	require.NoError(t, err)

	assert.Equal(t, expected, article)
}
