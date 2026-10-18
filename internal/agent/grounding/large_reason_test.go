// Purpose: Verify large reason prose survives native JSON decoding and grounding without clipping.
// Pattern: Boundary contract test.
// Usage: go test ./internal/agent/grounding -run TestLargeReason.
// Related: internal/agent/tool/reason.go, internal/agent/tool/intent.go.
package grounding

import (
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestLargeReason(t *testing.T) {
	reason := strings.Repeat("x", 17*1024*1024) + "\n근거"
	body, err := json.Marshal(map[string]string{"path": "file.txt", "reason": reason})
	if err != nil {
		t.Fatal(err)
	}
	call, got, err := tool.ExtractIntent(tool.Call{ID: "large", Name: "inspect_fixture", Arguments: body})
	if err != nil {
		t.Fatal(err)
	}
	record, err := Bind(call, got, "run")
	if err != nil || record.Reason != reason {
		t.Fatalf("large reason was rejected or changed: %v", err)
	}
	if err := tool.CheckReason(strings.Repeat("x", 32*1024*1024)); err != nil {
		t.Fatal(err)
	}
	if err := tool.CheckReason(strings.Repeat("x", 32*1024*1024+1)); err == nil {
		t.Fatal("accepted reason over 32 MiB")
	}
}
