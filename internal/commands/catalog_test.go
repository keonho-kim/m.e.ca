// Purpose: Verify command suggestions and parsing share the same five-command catalog.
// Pattern: Behavioral tests.
// Usage: go test ./internal/commands -run TestCommandSuggestions.
// Related: internal/commands/catalog.go, internal/commands/parse.go.
package commands

import "testing"

func TestCommandSuggestions(t *testing.T) {
	all := Suggestions("!")
	if len(all) != 5 {
		t.Fatalf("commands = %v", all)
	}
	for _, suggestion := range all {
		command, err := Parse(suggestion.Name)
		if err != nil || command.Kind == Message || suggestion.Description == "" {
			t.Fatalf("unusable suggestion: %+v, %v", suggestion, err)
		}
	}
	filtered := Suggestions("!mo")
	if len(filtered) != 1 || filtered[0].Name != "!model" {
		t.Fatalf("prefix filter: %v", filtered)
	}
	if len(Suggestions("!unknown")) != 0 {
		t.Fatal("unknown prefix matched a command")
	}
}
