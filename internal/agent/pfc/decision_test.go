// Purpose: Verify JSON-only PFC entry rejects ambiguous envelopes without changing ordinary replies.
// Pattern: Trust-boundary contract tests.
// Usage: go test ./internal/agent/pfc -run TestJSONEntry.
// Related: internal/agent/pfc/decision.go, internal/agent/run/action.go.
package pfc

import "testing"

func TestJSONEntry(t *testing.T) {
	for _, text := range []string{`{"tool":"read_file"}`, " \n{\n\t\"tool\" : \"read_file\"\n}\r\n"} {
		decision, err := ParseDecision(text)
		if err != nil || decision.Kind != Tool || decision.Name != "read_file" {
			t.Fatalf("%q: %+v %v", text, decision, err)
		}
	}
	for _, text := range []string{
		"TOOL read_file", "TOOL  read_file", "read_file",
		`{"tool":"read_file","path":"x"}`, `{"tool":"read_file","arguments":{}}`,
		`{"action":"pfc","tool":"read_file"}`, `{"tool":"read_file","tool":"write_file"}`,
		`{"tool":"read_file"} {"tool":"write_file"}`, `[{"tool":"read_file"}]`,
		"```json\n{\"tool\":\"read_file\"}\n```", "I will read:\n{\"tool\":\"read_file\"}",
		`{"tool":"read_file"} explanation`, `{"tool":null}`, `{"tool":12}`, `{"tool":""}`,
		`{"tool":"read file"}`, `{"tool":" read_file"}`, `{"tool":"Read_file"}`, `{}`, `{"tool":`,
	} {
		if decision, err := ParseDecision(text); err == nil {
			t.Fatalf("accepted %q: %+v", text, decision)
		}
	}
	decision, err := ParseDecision("DONE\n{\"tool\":\"read_file\"}")
	if err != nil || decision.Kind != Done || decision.Text != `{"tool":"read_file"}` {
		t.Fatalf("ordinary JSON answer became a tool: %+v %v", decision, err)
	}
}
