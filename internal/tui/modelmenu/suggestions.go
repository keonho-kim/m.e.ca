// Purpose: Find literal model IDs first and use ordered abbreviations when no literal matches exist.
// Pattern: Pure function.
// Usage: Model input refreshes suggestions from the current connection's discovered IDs.
// Related: internal/tui/session/model_form_keys.go.
package modelmenu

import (
	"github.com/keonho-kim/meca/internal/searchmatch"
	"strings"
)

func Suggestions(models []string, query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	var matches []string
	for _, model := range models {
		if strings.Contains(strings.ToLower(model), query) {
			matches = append(matches, model)
		}
	}
	if len(matches) == 0 && query != "" {
		for _, model := range models {
			if searchmatch.Subsequence(query, model) {
				matches = append(matches, model)
			}
		}
	}
	return matches
}
