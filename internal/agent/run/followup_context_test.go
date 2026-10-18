// Purpose: Verify follow-up decisions receive the same user scope and avoid repeating native execution records.
// Pattern: Runtime context contract tests.
// Usage: go test ./internal/agent/run -run TestFollowup.
// Related: internal/agent/run/work_context.go, internal/agent/run/task_context.go.
package run

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"testing"
)

func TestFollowupScopeReachesWorkerDecisions(t *testing.T) {
	previous := "Create a simple-server package using FastAPI, Pydantic and SQLAlchemy; empty files only for this first stage."
	j := &journal{messages: []modelapi.Message{{Role: "user", Content: previous}, {Role: "assistant", Content: "The scaffold exists."}}}
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "implementation"}), "Implemented the basic service."}}
	result, err := fixture(g, &executor{}, j).Run(context.Background(), "Now implement the basics", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("%+v %v", result, err)
	}
	for _, request := range g.requests {
		var body strings.Builder
		for _, m := range request.Messages {
			body.WriteString(m.Content)
		}
		if !strings.Contains(body.String(), previous) || !strings.Contains(body.String(), "Now implement the basics") {
			t.Fatal("worker lost referenced request")
		}
	}

}

func TestFollowupDoesNotDuplicateVisibleReceipts(t *testing.T) {
	previous := &Checkpoint{Objective: "Create a", Status: Stopped, Work: completion.State{Receipts: []completion.Receipt{{ID: "call", Tool: "inspect_fixture", Success: true, Output: "REPEATED_OUTPUT"}}}}
	s := execution{runner: Runner{PreviousRun: previous}, messages: []modelapi.Message{{Role: "tool", ToolCallID: "call", Content: "REPEATED_OUTPUT"}}}
	body, err := s.taskContext()
	if err != nil {
		t.Fatal(err)
	}
	var facts taskFacts
	if err = json.Unmarshal(body, &facts); err != nil {
		t.Fatal(err)
	}
	if len(facts.Results) != 0 {
		t.Fatal("native history duplicated in active facts")
	}
}

func TestFollowupDoesNotReinjectSupersededFileContent(t *testing.T) {
	s := execution{memoryContext: "Earlier scaffold", checkpoint: Checkpoint{Work: completion.State{Receipts: []completion.Receipt{
		{ID: "old", Tool: "inspect_fixture", Targets: []string{"app.py"}, Success: true, Stale: true, Output: "SUPERSEDED_STUB"},
		{ID: "new", Tool: "write_file", Targets: []string{"app.py"}, Success: true, Output: "Implemented app.py"},
	}}}}
	body, err := s.taskContext()
	if err != nil || strings.Contains(string(body), "SUPERSEDED_STUB") || !strings.Contains(string(body), "Implemented app.py") {
		t.Fatalf("superseded file content projected as current work: %s %v", body, err)
	}
}

func TestFollowupUsesCurrentResultsAfterResumedWork(t *testing.T) {
	s := execution{runner: Runner{PreviousRun: &Checkpoint{Objective: "Create a", Status: Stopped, Observation: "PREVIOUS_FAILURE", Work: completion.State{Receipts: []completion.Receipt{{ID: "old", Tool: "inspect_fixture", Success: true, Output: "PREVIOUS_FILE_STATE"}}}}}, checkpoint: Checkpoint{Observation: "Created a"}}
	body, err := s.taskContext()
	if err != nil || strings.Contains(string(body), "PREVIOUS_") || !strings.Contains(string(body), "Created a") {
		t.Fatalf("old run overrode current observations: %s %v", body, err)
	}
}
