// Package filter provides content filtering capabilities for the collection gateway.
package filter

import (
	"strings"
	"sync"
)

// KeywordFilter checks search queries against a blacklist of blocked keywords.
// It supports both exact match and substring (contains) match modes.
// All matching is case-insensitive. It is safe for concurrent use.
type KeywordFilter struct {
	mu       sync.RWMutex
	exact    map[string]struct{} // lowercased exact-match keywords
	contains []string            // lowercased substring-match keywords
}

// NewKeywordFilter creates a KeywordFilter pre-loaded with the given keywords.
// exactKeywords are matched exactly (case-insensitive).
// containsKeywords are matched as substrings (case-insensitive).
func NewKeywordFilter(exactKeywords, containsKeywords []string) *KeywordFilter {
	f := &KeywordFilter{
		exact: make(map[string]struct{}, len(exactKeywords)),
	}
	for _, kw := range exactKeywords {
		f.exact[strings.ToLower(kw)] = struct{}{}
	}
	for _, kw := range containsKeywords {
		f.contains = append(f.contains, strings.ToLower(kw))
	}
	return f
}

// IsBlocked returns true if the query matches any blocked keyword.
func (f *KeywordFilter) IsBlocked(query string) bool {
	if query == "" {
		return false
	}
	lower := strings.ToLower(query)

	f.mu.RLock()
	defer f.mu.RUnlock()

	// Check exact match.
	if _, ok := f.exact[lower]; ok {
		return true
	}

	// Check contains match.
	for _, kw := range f.contains {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}

// AddExact adds a keyword to the exact-match blacklist.
func (f *KeywordFilter) AddExact(keyword string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exact[strings.ToLower(keyword)] = struct{}{}
}

// AddContains adds a keyword to the contains-match blacklist.
func (f *KeywordFilter) AddContains(keyword string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.contains = append(f.contains, strings.ToLower(keyword))
}

// RemoveExact removes a keyword from the exact-match blacklist.
func (f *KeywordFilter) RemoveExact(keyword string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.exact, strings.ToLower(keyword))
}
