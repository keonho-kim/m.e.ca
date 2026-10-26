// Purpose: Verify literal model matches remain primary and abbreviations fill otherwise empty searches.
// Pattern: Pure search contract tests.
// Usage: go test ./internal/tui/settings/model -run TestSuggestions.
// Related: internal/tui/settings/model/suggestions.go.
package model

import (
	"slices"
	"testing"
)

func TestSuggestionsLiteralThenAbbreviation(t *testing.T) {
	models := []string{"vendor/Qwen-3.8-27B", "other/q38", "Gemma-4"}
	if got := suggestions(models, "Q38"); !slices.Equal(got, []string{"other/q38"}) {
		t.Fatal(got)
	}
	if got := suggestions(models[:1], "Q38"); !slices.Equal(got, models[:1]) {
		t.Fatal(got)
	}
	if got := suggestions(models, "gmm4"); !slices.Equal(got, []string{"Gemma-4"}) {
		t.Fatal(got)
	}
	if got := suggestions(models, ""); !slices.Equal(got, models) {
		t.Fatal(got)
	}
	if got := suggestions(models, "8qw"); len(got) != 0 {
		t.Fatal(got)
	}
}
