// Purpose: Match case-insensitive abbreviations while preserving character order.
// Pattern: Pure function using Unicode and string primitives.
// Usage: Command and model menus use this only when literal search has no results.
// Related: internal/commands/catalog.go, internal/tui/modelmenu/suggestions.go.
package searchmatch

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Subsequence matches each query character in order without requiring adjacency.
func Subsequence(query, value string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return false
	}
	want, size := utf8.DecodeRuneInString(query)
	for _, r := range value {
		if unicode.ToLower(r) != want {
			continue
		}
		query = query[size:]
		if query == "" {
			return true
		}
		want, size = utf8.DecodeRuneInString(query)
	}
	return false
}
