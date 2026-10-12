// Purpose: Verify small tool decisions and one typed argument at a time reject ambiguous input.
// Pattern: Trust-boundary behavior tests.
// Usage: go test ./internal/agent/dialogue.
// Related: internal/agent/dialogue/dialogue.go, internal/agent/tools.go.
package dialogue

import (
	"github.com/keonho-kim/v-code/internal/agent"
	"testing"
)

func TestDecisionGrammar(t *testing.T) {
	for _, tc := range []struct {
		text  string
		valid bool
	}{{"TOOL read_file", true}, {"CHECK read_file", true}, {"DONE\nReviewed the file", true}, {"WAIT\nWhich file?", true}, {"CONTINUE\nInspect next file", true}, {`{"name":"read_file","arguments":{"path":"x"}}`, false}, {"TOOL read_file\npath=x", false}, {"TOOL read_file write_file", false}, {"DONE", false}} {
		_, err := ParseDecision(tc.text)
		if (err == nil) != tc.valid {
			t.Fatalf("%q: %v", tc.text, err)
		}
	}
}
func TestArgumentPreservesLiteralTextAndRejectsAmbiguousNumbers(t *testing.T) {
	p := agent.ToolParameter{Name: "content", Kind: agent.Text, AllowEmpty: true, MaxBytes: 1024}
	v, err := ParseArgument(p, "  first\nsecond\n")
	if err != nil || string(v) != `"  first\nsecond\n"` {
		t.Fatalf("literal: %s %v", v, err)
	}
	n := agent.ToolParameter{Name: "limit", Kind: agent.Integer, Max: 100}
	for _, text := range []string{"two", "2 lines", "101", "-1", "{\"limit\":2}"} {
		if _, err := ParseArgument(n, text); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
	if v, err := ParseArgument(n, ""); err != nil || string(v) != "0" {
		t.Fatalf("optional default: %s %v", v, err)
	}
}

func TestPathCannotBeOneShotToolJSON(t *testing.T) {
	p := agent.ToolParameter{Name: "path", Kind: agent.LiteralPath, Required: true, MaxBytes: 4096}
	for _, text := range []string{`{"path":"src/main.go"}`, `"src/main.go"`, "src/\nmain.go"} {
		if _, err := ParseArgument(p, text); err == nil {
			t.Fatalf("accepted path wrapper %q", text)
		}
	}
	if _, err := ParseArgument(p, "src/my components/main.go"); err != nil {
		t.Fatal(err)
	}
}
