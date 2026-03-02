// Package i18n provides language detection, context propagation, and
// Accept-Language header parsing for the collection-gateway API.
package i18n

import (
	"context"
	"net/http"
	"strings"
)

// Lang represents a BCP-47 language tag supported by the gateway.
type Lang string

// Supported language constants.
const (
	LangZhTW Lang = "zh-TW"
	LangJA   Lang = "ja"
	LangEN   Lang = "en"
)

// DefaultLang is the fallback language when no match is found.
var DefaultLang = LangZhTW

// SupportedLangs lists every language the gateway can serve.
var SupportedLangs = []Lang{LangZhTW, LangJA, LangEN}

// contextKey is an unexported type used as the context key for Lang values
// to avoid collisions with other packages.
type contextKey struct{}

// WithLang returns a copy of ctx that carries the given Lang.
func WithLang(ctx context.Context, lang Lang) context.Context {
	return context.WithValue(ctx, contextKey{}, lang)
}

// FromContext extracts the Lang stored in ctx. If no Lang is present it
// returns DefaultLang.
func FromContext(ctx context.Context) Lang {
	if lang, ok := ctx.Value(contextKey{}).(Lang); ok {
		return lang
	}
	return DefaultLang
}

// FromRequest extracts a Lang from the request's Accept-Language header.
func FromRequest(r *http.Request) Lang {
	return ParseAcceptLanguage(r.Header.Get("Accept-Language"))
}

// ParseAcceptLanguage parses an Accept-Language header value and returns
// the best matching supported Lang. It iterates tags from left to right
// (highest priority first), strips quality suffixes, and performs
// case-insensitive matching. If no tag matches, DefaultLang is returned.
func ParseAcceptLanguage(header string) Lang {
	header = strings.TrimSpace(header)
	if header == "" {
		return DefaultLang
	}

	for _, part := range strings.Split(header, ",") {
		// Strip quality value (;q=...)
		if idx := strings.IndexByte(part, ';'); idx != -1 {
			part = part[:idx]
		}
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}

		if lang, ok := matchTag(tag); ok {
			return lang
		}
	}

	return DefaultLang
}

// matchTag performs case-insensitive matching of a single BCP-47 language
// tag against the supported languages.
func matchTag(tag string) (Lang, bool) {
	lower := strings.ToLower(tag)

	switch {
	// Traditional Chinese variants
	case lower == "zh-tw" || lower == "zh-hant" || lower == "zh":
		return LangZhTW, true

	// Japanese (exact or any sub-tag like ja-JP)
	case lower == "ja" || strings.HasPrefix(lower, "ja-"):
		return LangJA, true

	// English (exact or any sub-tag like en-US)
	case lower == "en" || strings.HasPrefix(lower, "en-"):
		return LangEN, true
	}

	return "", false
}
