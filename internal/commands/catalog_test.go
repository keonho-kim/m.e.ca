// Purpose: Verify command suggestions and parsing share the same local command catalog.
// Pattern: Behavioral tests.
// Usage: go test ./internal/commands -run TestCommandSuggestions.
// Related: internal/commands/catalog.go, internal/commands/parse.go.
package commands

import "testing"

func TestCompactionBrowserCommand(t *testing.T) {
	action, argument, err := BrowserInput(" !compact ")
	if err != nil || action != "compact" || argument != "" {
		t.Fatalf("compaction command = %q %q, %v", action, argument, err)
	}
	if _, _, err := BrowserInput("!compact more"); err == nil {
		t.Fatal("compaction accepted an argument")
	}
}

func TestCommandSuggestions(t *testing.T) {
	all := Suggestions("!", describeCommand)
	seen := make(map[string]bool)
	for _, suggestion := range all {
		command, err := Parse(suggestion.Name)
		if err != nil || command.Kind == Message || suggestion.Description == "" || seen[suggestion.Name] {
			t.Fatalf("unusable suggestion: %+v, %v", suggestion, err)
		}
		seen[suggestion.Name] = true
	}
	for _, name := range []string{"!new", "!model", "!permission", "!team", "!language", "!dir", "!skills", "!compact", "!quit"} {
		if !seen[name] {
			t.Fatalf("supported command is not discoverable: %s", name)
		}
	}
	filtered := Suggestions("!mo", describeCommand)
	if len(filtered) != 1 || filtered[0].Name != "!model" {
		t.Fatalf("prefix filter: %v", filtered)
	}
	if len(Suggestions("!unknown", describeCommand)) != 0 {
		t.Fatal("unknown prefix matched a command")
	}
	if command, err := Parse("!permission"); err != nil || command.Kind != Permission || len(Suggestions("!permission", describeCommand)) != 1 {
		t.Fatal("permission command is unavailable")
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
		{"!lang", []string{"!language"}},
		{"!언어", []string{"!language"}},
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

func TestCommandAbbreviationFallback(t *testing.T) {
	for _, tc := range []struct{ query, want string }{{"!mdl", "!model"}, {"!lng", "!language"}, {"!qt", "!quit"}} {
		got := Suggestions(tc.query, describeCommand)
		if len(got) != 1 || got[0].Name != tc.want {
			t.Fatalf("%s: %+v", tc.query, got)
		}
	}
}
