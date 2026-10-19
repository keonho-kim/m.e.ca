// Purpose: Verify command suggestions and parsing share the same five-command catalog.
// Pattern: Behavioral tests.
// Usage: go test ./internal/commands -run TestCommandSuggestions.
// Related: internal/commands/catalog.go, internal/commands/parse.go.
package commands

import "testing"

func TestCommandSuggestions(t *testing.T) {
	all := Suggestions("!", describeCommand)
	if len(all) != 5 {
		t.Fatalf("commands = %v", all)
	}
	for _, suggestion := range all {
		command, err := Parse(suggestion.Name)
		if err != nil || command.Kind == Message || suggestion.Description == "" {
			t.Fatalf("unusable suggestion: %+v, %v", suggestion, err)
		}
	}
	filtered := Suggestions("!mo", describeCommand)
	if len(filtered) != 1 || filtered[0].Name != "!model" {
		t.Fatalf("prefix filter: %v", filtered)
	}
	if len(Suggestions("!unknown", describeCommand)) != 0 {
		t.Fatal("unknown prefix matched a command")
	}
}

func TestCommandSearchMatchesNamesAndDescriptions(t *testing.T) {
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"!mo", []string{"!model"}},
		{"!odel", []string{"!model"}},
		{"!모델", []string{"!model"}},
		{"!추론", []string{"!model"}},
		{"!새", []string{"!new"}},
		{"!d", []string{"!dir", "!model"}},
		{"!unknown", nil},
		{"model", nil},
	} {
		t.Run(tc.query, func(t *testing.T) {
			got := Suggestions(tc.query, describeCommand)
			if len(got) != len(tc.want) {
				t.Fatalf("%q: %v", tc.query, got)
			}
			for i, name := range tc.want {
				if got[i].Name != name {
					t.Fatalf("%q: %v", tc.query, got)
				}
			}
		})
	}
}

func describeCommand(description string) string { return description }
