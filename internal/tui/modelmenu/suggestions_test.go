// Purpose: Verify literal model matches remain primary and abbreviations fill otherwise empty searches.
// Pattern: Pure search contract tests.
// Usage: go test ./internal/tui/modelmenu -run TestSuggestions.
// Related: internal/tui/modelmenu/suggestions.go.
package modelmenu

import (
	"slices"
	"testing"
)

func TestSuggestionsLiteralThenAbbreviation(t *testing.T) {
	models := []string{"vendor/Qwen-3.8-27B", "other/q38", "Gemma-4"}
	if got := Suggestions(models, "Q38"); !slices.Equal(got, []string{"other/q38"}) {
		t.Fatal(got)
	}
	if got := Suggestions(models[:1], "Q38"); !slices.Equal(got, models[:1]) {
		t.Fatal(got)
	}
	if got := Suggestions(models, "gmm4"); !slices.Equal(got, []string{"Gemma-4"}) {
		t.Fatal(got)
	}
	if got := Suggestions(models, ""); !slices.Equal(got, models) {
		t.Fatal(got)
	}
	if got := Suggestions(models, "8qw"); len(got) != 0 {
		t.Fatal(got)
	}
}
