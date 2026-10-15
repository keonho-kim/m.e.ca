// Purpose: Verify intent remains bound to one exact operation without generated bullet parsing.
// Pattern: Trust-boundary tests.
// Usage: go test ./internal/agent/grounding.
// Related: internal/agent/grounding/record.go.
package grounding

import (
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestRecordBindsExactCall(t *testing.T) {
	call := tool.Call{ID: "c", Name: "read_file", Arguments: []byte(`{"path":"a"}`)}
	record, err := Bind(call, "Inspect requested file.", "run_1")
	if err != nil || !record.Matches(call) {
		t.Fatalf("%+v %v", record, err)
	}
	call.Arguments = []byte(`{"path":"b"}`)
	if record.Matches(call) {
		t.Fatal("intent reused for another operation")
	}
	for _, reason := range []string{"", "line\nbreak", strings.Repeat("x", 257)} {
		if _, err := Bind(call, reason, "run_1"); err == nil {
			t.Fatalf("accepted %q", reason)
		}
	}
}
