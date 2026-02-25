// Package brand provides a brand registry and a multi-level brand
// identification pipeline for the Rakutao collection gateway.
package brand

import (
	"strings"
	"time"
	"unicode"
)

// Entry represents a single brand in the registry.
type Entry struct {
	ID         string    `json:"id"`
	NameStd    string    `json:"name_std"`
	NameJA     string    `json:"name_ja"`
	NameZhTW   string    `json:"name_zh_tw"`
	Aliases    []string  `json:"aliases"`
	Category   string    `json:"category"`
	LogoURL    string    `json:"logo_url"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
}

// Registry holds an indexed collection of brand entries for fast lookup.
type Registry struct {
	entries  []Entry
	aliasMap map[string]*Entry
}

// NewRegistry creates an empty brand registry.
func NewRegistry() *Registry {
	return &Registry{
		entries:  make([]Entry, 0),
		aliasMap: make(map[string]*Entry),
	}
}

// Add indexes a brand entry by its standard name, Japanese name, Chinese name,
// and all aliases. Latin strings are lowercased for case-insensitive lookup;
// Japanese/CJK strings are stored as-is.
func (r *Registry) Add(e Entry) {
	r.entries = append(r.entries, e)
	ptr := &r.entries[len(r.entries)-1]

	// Index standard name (lowercased).
	if e.NameStd != "" {
		r.aliasMap[strings.ToLower(e.NameStd)] = ptr
	}

	// Index Japanese name (exact).
	if e.NameJA != "" {
		r.aliasMap[e.NameJA] = ptr
	}

	// Index Traditional Chinese name (exact).
	if e.NameZhTW != "" {
		r.aliasMap[e.NameZhTW] = ptr
	}

	// Index all aliases.
	for _, alias := range e.Aliases {
		key := normalizeKey(alias)
		r.aliasMap[key] = ptr
	}
}

// FindByAlias performs a case-insensitive (for Latin) alias lookup.
func (r *Registry) FindByAlias(alias string) *Entry {
	key := normalizeKey(alias)
	return r.aliasMap[key]
}

// MatchInText scans the given text for any known brand alias and returns the
// first match found. It checks all indexed keys against the text.
func (r *Registry) MatchInText(text string) *Entry {
	if text == "" {
		return nil
	}

	textLower := strings.ToLower(text)

	// Iterate over all aliasMap keys and check if the text contains them.
	// We try longer keys first for better matching, but for simplicity we
	// iterate in map order and return the first hit.
	for key, entry := range r.aliasMap {
		// For CJK/Japanese keys, match against original text.
		if isCJK(key) {
			if strings.Contains(text, key) {
				return entry
			}
		} else {
			// For Latin keys, match case-insensitively.
			if strings.Contains(textLower, key) {
				return entry
			}
		}
	}

	return nil
}

// All returns a copy of all entries in the registry.
func (r *Registry) All() []Entry {
	result := make([]Entry, len(r.entries))
	copy(result, r.entries)
	return result
}

// normalizeKey returns a lookup key: lowercased for Latin strings, exact for
// CJK/Japanese strings.
func normalizeKey(s string) string {
	if isCJK(s) {
		return s
	}
	return strings.ToLower(s)
}

// isCJK reports whether the string contains any CJK or Japanese characters.
func isCJK(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) ||
			unicode.Is(unicode.Katakana, r) ||
			unicode.Is(unicode.Hiragana, r) {
			return true
		}
	}
	return false
}
