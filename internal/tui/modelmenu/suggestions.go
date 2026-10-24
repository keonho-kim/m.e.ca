// Purpose: Filter discovered model IDs by case-insensitive substring without changing their spelling.
// Pattern: Pure function.
// Usage: Model input refreshes suggestions from the current connection's discovered IDs.
// Related: internal/tui/session/model_form_keys.go.
package modelmenu

import "strings"

func Suggestions(models []string, query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	var matches []string
	for _, model := range models {
		if strings.Contains(strings.ToLower(model), query) {
			matches = append(matches, model)
		}
	}
	return matches
}
