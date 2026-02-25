package search

import (
	"context"
	"regexp"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// jaPattern matches any Hiragana or Katakana character, which indicates the
// text contains Japanese script.
var jaPattern = regexp.MustCompile(`[\x{3040}-\x{309F}\x{30A0}-\x{30FF}]`)

// Translator translates a keyword from a source language into Japanese.
type Translator interface {
	Translate(ctx context.Context, keyword, sourceLang string) (string, error)
}

// Gateway orchestrates search operations including keyword translation,
// query building, and result aggregation.
type Gateway struct {
	translator Translator
}

// NewGateway creates a new Gateway with the given Translator.
func NewGateway(translator Translator) *Gateway {
	return &Gateway{
		translator: translator,
	}
}

// prepareQuery inspects the query keyword to detect whether it is already
// Japanese (contains hiragana/katakana). If the keyword is Japanese it is
// assigned directly to KeywordJA. Otherwise the Translator is invoked to
// produce a Japanese translation. On translation error the original keyword
// is used as a fallback so that searches never fail purely because of
// translation issues.
func (g *Gateway) prepareQuery(ctx context.Context, q domain.SearchQuery) (domain.SearchQuery, error) {
	if q.Keyword == "" {
		return q, nil
	}

	if isJapanese(q.Keyword) {
		q.KeywordJA = q.Keyword
		return q, nil
	}

	translated, err := g.translator.Translate(ctx, q.Keyword, q.UserLang)
	if err != nil {
		// Fallback: use the original keyword so searches are never blocked
		// by translation failures.
		q.KeywordJA = q.Keyword
		return q, nil
	}

	q.KeywordJA = translated
	return q, nil
}

// isJapanese reports whether s contains any Hiragana or Katakana characters.
func isJapanese(s string) bool {
	return jaPattern.MatchString(s)
}
