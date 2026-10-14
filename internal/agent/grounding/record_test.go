// Purpose: Verify concise self-grounding records bind evidence to one exact PFC call.
// Pattern: Trust-boundary tests.
// Usage: go test ./internal/agent/grounding.
// Related: internal/agent/grounding/record.go, internal/agent/run/grounding.go.
package grounding

import (
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"testing"
)

func TestRecordBindsExactCall(t *testing.T) {
	call := tool.Call{ID: "call", Name: "read_file", Arguments: json.RawMessage(`{"path":"README.md"}`)}
	record, err := Parse("- Evidence: User named README.md; contents are unknown.\n- Action: Inspect the requested file.\n- Verify: Confirm the relevant documented behavior.", call)
	if err != nil || !record.Matches(call) {
		t.Fatalf("record=%+v err=%v", record, err)
	}
	call.Arguments = json.RawMessage(`{"path":"secret"}`)
	if record.Matches(call) {
		t.Fatal("grounding reused for different arguments")
	}
	for _, invalid := range []string{"", `{"evidence":"guessed"}`, "- Evidence: x\n- Action: x", "- Evidence: x\n- Action: x\n- Verify: "} {
		if _, err := Parse(invalid, call); err == nil {
			t.Fatalf("invalid grounding accepted: %q", invalid)
		}
	}
}
