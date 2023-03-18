package chatgpt

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/Semior001/newsfeed/app/store"
	"github.com/go-shiori/go-readability"
)

// Extractor is extracts article from HTML page.
type Extractor struct {
	parser readability.Parser
}

// NewExtractor creates new Extractor.
func NewExtractor(debug bool) Extractor {
	svc := Extractor{parser: readability.NewParser()}
	svc.parser.Debug = debug

	return svc
}

// Extract extracts article from an HTML page.
func (e Extractor) Extract(rd io.Reader) (store.Article, error) {
	doc, err := readability.FromReader(rd, nil)
	if err != nil {
		return store.Article{}, fmt.Errorf("parse html: %w", err)
	}

	return store.Article{
		Title:    doc.Title,
		Excerpt:  doc.Excerpt,
		Content:  e.sanitize(doc.TextContent),
		Author:   doc.Byline,
		ImageURL: doc.Image,
	}, nil
}

func (e Extractor) sanitize(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	// nbsp
	s = strings.ReplaceAll(s, "\u00a0", " ")

	re := regexp.MustCompile(`\s+`)
	sanitized := re.ReplaceAllString(s, " ")

	return sanitized
}
