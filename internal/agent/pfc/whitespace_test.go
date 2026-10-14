// Purpose: Verify typed PFC whitespace normalization preserves literal content and frozen arguments.
// Pattern: Parser boundary tests.
// Usage: go test ./internal/agent/pfc -run TestWhitespace.
// Related: internal/agent/pfc/pfc.go, internal/agent/run/arguments.go, internal/tools/local/arguments.go.
package pfc

import (
	"encoding/json"
	"testing"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func TestWhitespacePathBoundaryLines(t *testing.T) {
	parameter := toolapi.Parameter{Name: "path", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
	for _, test := range []struct{ input, want string }{
		{"\n\nAGENTS.md\n\n", "AGENTS.md"},
		{"\r\nAGENTS.md\r\n", "AGENTS.md"},
		{" \t\r\n\n src/my file \r\n\t ", " src/my file "},
		{" src/my file ", " src/my file "},
	} {
		value, err := ParseArgument(parameter, test.input)
		if err != nil {
			t.Fatalf("%q: %v", test.input, err)
		}
		var got string
		if err := json.Unmarshal(value, &got); err != nil || got != test.want {
			t.Fatalf("%q: got %q, want %q; %v", test.input, got, test.want, err)
		}
		if err := ValidateArgument(parameter, value); err != nil {
			t.Fatalf("normalized path rejected at execution: %v", err)
		}
	}
	for _, input := range []string{"\nfirst\nsecond\n", "\n \t\r\n", "\n{\"path\":\"a\"}\n"} {
		if _, err := ParseArgument(parameter, input); err == nil {
			t.Fatalf("accepted ambiguous path %q", input)
		}
	}
}

func TestWhitespaceFrozenPathCannotChangeDuringValidation(t *testing.T) {
	parameter := toolapi.Parameter{Name: "path", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
	for _, input := range []string{"\na", "a\n", " \n a \n "} {
		value, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateArgument(parameter, value); err == nil {
			t.Fatalf("execution accepted unnormalized path %q", input)
		}
	}
}

func TestWhitespaceLiteralContentIsPreserved(t *testing.T) {
	for _, name := range []string{"content", "new_text", "pattern", "command"} {
		parameter := toolapi.Parameter{Name: name, Kind: toolapi.Text, AllowEmpty: true, MaxBytes: 1024}
		input := "\r\n\n  keep indentation \t\n\n"
		value, err := ParseArgument(parameter, input)
		if err != nil {
			t.Fatal(err)
		}
		var got string
		if err := json.Unmarshal(value, &got); err != nil || got != input {
			t.Fatalf("%s lost literal whitespace: %q %v", name, got, err)
		}
	}
}

func TestWhitespaceScalarValues(t *testing.T) {
	for _, test := range []struct {
		parameter   toolapi.Parameter
		input, want string
	}{
		{toolapi.Parameter{Name: "limit", Kind: toolapi.Integer, Max: 100}, "\n\t 2 \r\n", "2"},
		{toolapi.Parameter{Name: "start_anchor", Kind: toolapi.LineAnchor, MaxBytes: 32}, "\n\t42#9a72d30f \r\n", `"42#9a72d30f"`},
	} {
		value, err := ParseArgument(test.parameter, test.input)
		if err != nil || string(value) != test.want {
			t.Fatalf("%s: %s %v", test.parameter.Name, value, err)
		}
	}
}
