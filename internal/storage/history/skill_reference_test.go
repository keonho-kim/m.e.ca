// Purpose: Check preservation of skill reference evidence in compaction inputs.
// Pattern: Behavior regression test.
// Usage: go test ./internal/storage/history.
// Related: internal/storage/history/compaction.go.
package history

import (
	"github.com/keonho-kim/meca/internal/agent/model"
	"strings"
	"testing"
)

func TestSkillReferenceCompaction(t *testing.T) {
	records, err := Compaction([]Record{{Sequence: 1, Kind: "message", Message: model.Message{Role: "tool", ToolCallID: "call_1", ToolName: "read_skill_file", Content: `{"text":"reference-instruction-canary","media_type":"text/plain","offset":0,"next_offset":0,"total_bytes":28}`}}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range records {
		found = found || strings.Contains(r.Text, "reference-instruction-canary")
	}
	if !found {
		t.Fatalf("reference read vanished from compaction input: %#v", records)
	}
}
