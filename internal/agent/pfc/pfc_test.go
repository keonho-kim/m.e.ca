// Purpose: Verify small tool decisions and one typed argument at a time reject ambiguous input.
// Pattern: Trust-boundary behavior tests.
// Usage: go test ./internal/agent/pfc.
// Related: internal/agent/pfc/pfc.go, internal/agent/tool/execution.go.
package pfc

import (
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
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
	p := toolapi.Parameter{Name: "content", Kind: toolapi.Text, AllowEmpty: true, MaxBytes: 1024}
	v, err := ParseArgument(p, "  first\nsecond\n")
	if err != nil || string(v) != `"  first\nsecond\n"` {
		t.Fatalf("literal: %s %v", v, err)
	}
	n := toolapi.Parameter{Name: "limit", Kind: toolapi.Integer, Max: 100}
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
	p := toolapi.Parameter{Name: "path", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
	for _, text := range []string{`{"path":"src/main.go"}`, `"src/main.go"`, "src/\nmain.go"} {
		if _, err := ParseArgument(p, text); err == nil {
			t.Fatalf("accepted path wrapper %q", text)
		}
	}
	if _, err := ParseArgument(p, "src/my components/main.go"); err != nil {
		t.Fatal(err)
	}
}
func TestPathAcceptsALineEndingWithoutChangingItsValue(t *testing.T) {
	p := toolapi.Parameter{Name: "path", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
	for _, reply := range []string{"README.md\n", "README.md\r\n"} {
		value, err := ParseArgument(p, reply)
		if err != nil || string(value) != `"README.md"` {
			t.Fatalf("path line ending: %s %v", value, err)
		}
	}
}

func TestLineAnchorFieldIsLiteralAndValidatedBeforePreparation(t *testing.T) {
	parameter := toolapi.Parameter{Name: "start_anchor", Kind: toolapi.LineAnchor, Required: true, MaxBytes: 32}
	value, err := ParseArgument(parameter, "42#9a72d30f\n")
	if err != nil || string(value) != `"42#9a72d30f"` {
		t.Fatalf("value=%s err=%v", value, err)
	}
	for _, text := range []string{`{"start_anchor":"42#9a72d30f"}`, "42#9a72", "42#zzzzzzzz", "0#9a72d30f"} {
		if _, err := ParseArgument(parameter, text); err == nil {
			t.Fatalf("invalid anchor accepted: %q", text)
		}
	}
}
