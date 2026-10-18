// Purpose: Verify complete argument objects preserve literals and reject ambiguous or invalid fields.
// Pattern: Tool boundary contract tests.
// Usage: go test ./internal/agent/tool.
// Related: internal/agent/tool/arguments.go, internal/agent/tool/json_schema.go.
package tool

import (
	"encoding/json"
	"testing"
)

func TestNativeArgumentsValidateAndPreserveLiterals(t *testing.T) {
	definition := Definition{Name: "write_file", Parameters: []Parameter{
		{Name: "path", Kind: LiteralPath, Required: true, MaxBytes: 4096},
		{Name: "content", Kind: Text, Required: true, AllowEmpty: true, MaxBytes: 1024},
	}}
	body := json.RawMessage(`{"path":" file ","content":"\n  code\n\n"}`)
	values, err := ValidateArguments(definition, body)
	if err != nil {
		t.Fatal(err)
	}
	var content string
	if err := json.Unmarshal(values["content"], &content); err != nil || content != "\n  code\n\n" {
		t.Fatalf("content=%q err=%v", content, err)
	}
	for _, invalid := range []string{
		`{"path":"a","path":"b","content":""}`,
		`{"path":1,"content":""}`,
		`{"path":"a","content":null}`,
		`{"path":"a"}`,
		`{"path":"a","content":"","unexpected":"value"}`,
		`{"path":"a\nb","content":""}`,
		`{"path":"a","content":""} {}`,
	} {
		if _, err := ValidateArguments(definition, json.RawMessage(invalid)); err == nil {
			t.Fatalf("accepted %s", invalid)
		}
	}
}
