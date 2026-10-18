// Purpose: Verify compaction uses concise outcomes while leaving source history unchanged.
// Pattern: Projection contract tests.
// Usage: go test ./internal/storage/history -run TestCompaction.
// Related: internal/storage/history/compaction.go.
package history

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent/grounding"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
)

func TestCompactionOmitsDuplicateAuditRecords(t *testing.T) {
	page := []Record{
		{Sequence: 1, Kind: "message", Message: modelapi.Message{Role: "user", Content: "Preserve this exact request"}},
		{Sequence: 2, Kind: "event", Event: run.Event{Kind: "monitor", Monitor: &monitor.Record{Detail: "duplicate audit"}}},
		{Sequence: 3, Kind: "event", Event: run.Event{Kind: "tool_proposed"}},
		{Sequence: 4, Kind: "event", Event: run.Event{Kind: "self_grounding", Grounding: &grounding.Record{}}},
		{Sequence: 5, Kind: "event", Event: run.Event{Kind: "tool_failed", Operation: &tool.Operation{Name: "execute_command", Command: "go test ./...", Status: "tool_failed", Process: &tool.ProcessOutcome{Status: "exited", ExitCode: 1, Diagnostic: "parser failed"}, Changes: []tool.FileChange{{Action: "edit_file", Path: "a.go"}}}, Output: json.RawMessage(`{"stdout":"raw large log"}`)}},
		{Sequence: 6, Kind: "message", Message: modelapi.Message{Role: "tool", Content: `{"Content":"important source excerpt","Digest":"irrelevant hash","anchors":["metadata"]}`}},
	}
	original, _ := json.Marshal(page)
	records, err := Compaction(page)
	if err != nil || len(records) != 3 {
		t.Fatalf("%+v %v", records, err)
	}
	body, _ := json.Marshal(records)
	for _, want := range []string{"Preserve this exact request", "exit=1", "parser failed", "a.go", "important source excerpt"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %s", want)
		}
	}
	for _, unwanted := range []string{"duplicate audit", "raw large log", "irrelevant hash", "anchors"} {
		if strings.Contains(string(body), unwanted) {
			t.Errorf("unnecessary source: %s", unwanted)
		}
	}
	after, _ := json.Marshal(page)
	if string(original) != string(after) {
		t.Fatal("projection changed original records")
	}
}
