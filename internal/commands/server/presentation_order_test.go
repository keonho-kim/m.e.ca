// Purpose: Keep generation and activity publication positions stable through snapshot recovery.
// Pattern: Deterministic server stream contract tests.
// Usage: go test ./internal/commands/server -run TestPresentationOrder.
// Related: internal/commands/server/stream_projection.go, internal/commands/server/activity.go.
package server

import (
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"testing"
)

func TestPresentationOrderSurvivesSnapshots(t *testing.T) {
	s, r := projectionFixture()
	s.emit(r, record.Event{Presentation: "request.00000000000000000001", Kind: "generation_started", Agent: "z"})
	s.emit(r, record.Event{Kind: "stream", Agent: "z", Thinking: "first"})
	first := s.State().Drafts[0].StartSequence
	s.emit(r, record.Event{Kind: "tool_started", Agent: "z", Call: &tool.Call{ExecutionID: "one", Name: "load_skill"}})
	s.emit(r, record.Event{Kind: "generation_started", Agent: "a"})
	s.emit(r, record.Event{Kind: "stream", Agent: "a", Thinking: "later"})
	snapshot := s.State()
	if first == 0 || len(snapshot.Activity) != 1 || snapshot.Activity[0].Sequence <= first {
		t.Fatalf("invalid order: %+v", snapshot)
	}
	for _, d := range snapshot.Drafts {
		if d.Agent == "a" && d.StartSequence <= snapshot.Activity[0].Sequence {
			t.Fatalf("new generation precedes tool: %+v", d)
		}
		if d.Agent == "z" && (d.StartSequence != first || d.Presentation != "request.00000000000000000001") {
			t.Fatal("content growth changed start sequence")
		}
	}
}

func TestApprovalReasonsAreDetachedAndBounded(t *testing.T) {
	original := &tool.Approval{Reasons: []string{"Outside workspace"}}
	projected := ProjectEvent(Event{Record: record.Event{Approval: original}})
	original.Reasons[0] = "changed"
	if projected.Record.Approval.Reasons[0] != "Outside workspace" {
		t.Fatal("projection aliases approval reasons")
	}
}
