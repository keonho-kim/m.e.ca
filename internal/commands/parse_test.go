// Purpose: Verify local command recognition and quoted directory arguments.
// Pattern: Table-driven behavioral tests.
// Usage: go test ./internal/commands.
// Related: internal/commands/parse.go, SYSTEM.md.

package commands

import "testing"

func TestParse(t *testing.T) {
	for _, test := range []struct {
		input    string
		kind     Kind
		argument string
	}{
		{"   ", Empty, ""},
		{"!model", Model, ""},
		{"!new", NewSession, ""},
		{"!dir", Directory, ""},
		{`!dir "src/my components/"`, Directory, "src/my components/"},
		{`!dir '~/Downloads/my files'`, Directory, "~/Downloads/my files"},
		{`:q`, Quit, ""},
		{"!quit", Exit, ""},
		{"explain !model in this sentence", Message, "explain !model in this sentence"},
	} {
		t.Run(test.input, func(t *testing.T) {
			command, err := Parse(test.input)
			if err != nil || command.Kind != test.kind || command.Argument != test.argument {
				t.Fatalf("Parse(%q) = %+v, %v", test.input, command, err)
			}
		})
	}
}

func TestParseRejectsUnknownOrAmbiguousCommands(t *testing.T) {
	for _, input := range []string{"!unknown", "!model extra", "!new extra", "!quit extra", "!dir a b", `!dir "unterminated`, "!dir a\\", "!dir\x00secret"} {
		if _, err := Parse(input); err == nil {
			t.Errorf("accepted invalid command %q", input)
		}
	}
}
