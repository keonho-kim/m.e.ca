// Purpose: Verify native intent and persisted grounding accept the same generous prose contract.
// Pattern: Boundary contract test.
// Usage: go test ./internal/agent/grounding -run TestLongReason.
// Related: internal/agent/tool/intent.go, internal/agent/grounding/record.go.
package grounding

import (
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestLongReasonPreservesMultilineSource(t *testing.T) {
	reason := strings.Repeat("문맥을 확인합니다. ", 500) + "\n다음 작업의 근거입니다."
	body, _ := json.Marshal(map[string]string{"path": "a", "reason": reason})
	call, text, err := tool.ExtractIntent(tool.Call{ID: "c", Name: "inspect_fixture", Arguments: body})
	if err != nil {
		t.Fatal(err)
	}
	record, err := Bind(call, text, "run")
	if err != nil || record.Reason != reason {
		t.Fatalf("reason changed: %v", err)
	}
}
