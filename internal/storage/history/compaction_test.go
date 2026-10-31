// Purpose: Verify compaction uses concise outcomes while leaving source history unchanged.
// Pattern: Projection contract tests.
// Usage: go test ./internal/storage/history -run TestCompaction.
// Related: internal/storage/history/compaction.go.
package history

import (
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestCompactionOmitsDuplicateAuditRecords(t *testing.T) {
	page := []Record{
		{Sequence: 1, Kind: "message", Message: modelapi.Message{Role: "user", Content: "Preserve this exact request"}},
		{Sequence: 3, Kind: "event", Event: runrecord.Event{Kind: "tool_proposed"}},
		{Sequence: 5, Kind: "event", Event: runrecord.Event{Kind: "tool_failed", Operation: &tool.Operation{Name: "execute_command", Command: "go test ./...", Status: "tool_failed", Process: &tool.ProcessOutcome{Status: "exited", ExitCode: 1, Diagnostic: "parser failed"}, Changes: []tool.FileChange{{Action: "edit_file", Path: "a.go"}}}, Output: json.RawMessage(`{"stdout":"raw large log"}`)}},
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

func TestCompactionDistinguishesUnfinishedClaimsFromOutcomes(t *testing.T) {
	records, err := Compaction([]Record{
		{Sequence: 1, Kind: "message", Message: modelapi.Message{Role: "assistant", Content: "All files have been created", Incomplete: true, Reasoning: "unfinished thinking"}},
		{Sequence: 2, Kind: "event", Event: runrecord.Event{Kind: "thinking", Thinking: "unfinished thinking"}},
		{Sequence: 3, Kind: "event", Event: runrecord.Event{Kind: "tool_failed", Text: "write failed: permission denied"}},
		{Sequence: 4, Kind: "message", Message: modelapi.Message{Role: "user", Content: "Create only README.md now"}},
	})
	if err != nil || len(records) != 3 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if records[0].Role != "unfinished_assistant_draft" || records[0].Text != "All files have been created" || records[1].Role != "failure" || records[2].Role != "user" {
		t.Fatalf("draft promoted to outcome or evidence lost: %+v", records)
	}
	body, _ := json.Marshal(records)
	if strings.Contains(string(body), "unfinished thinking") {
		t.Fatal("unfinished thinking reached compaction")
	}
}

func TestCompactionRetainsAdvisoryReviewWithoutDuplicateInstructions(t *testing.T) {
	records, err := Compaction([]Record{
		{Sequence: 1, Kind: "message", Message: modelapi.Message{Role: "system", Content: "Review observations: missing endpoint"}},
		{Sequence: 2, Kind: "event", Event: runrecord.Event{Kind: "lat_feedback", Text: "api.py: missing requested endpoint; add its handler"}},
		{Sequence: 3, Kind: "event", Event: runrecord.Event{Kind: "lat_unavailable", Text: "transport unavailable"}},
		{Sequence: 4, Kind: "event", Event: runrecord.Event{Kind: "lat_clean"}},
	})
	if err != nil || len(records) != 1 || records[0].Role != "review_observation" || !strings.Contains(records[0].Text, "missing requested endpoint") {
		t.Fatalf("pending review context lost or duplicated: %+v %v", records, err)
	}
}
