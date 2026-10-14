// Purpose: Verify missing self-grounding blocks preparation and valid records remain in subsequent context.
// Pattern: Task state-machine contract tests.
// Usage: go test ./internal/agent/run -run TestGrounding.
// Related: internal/agent/run/grounding.go, internal/agent/run/execute.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"testing"
)

type ungrounded struct{ base *generator }

func (g ungrounded) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	if strings.Contains(r.Messages[len(r.Messages)-1].Content, "\nSELF-GROUNDING:") {
		return emit(modelapi.Event{Delta: "I know the answer", Done: true})
	}
	return g.base.Generate(ctx, r, emit)
}

func TestGroundingMandatoryAndBound(t *testing.T) {
	g := &generator{replies: []string{"TOOL read_file", "README.md"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Generator = ungrounded{g}
	if _, err := r.Run(context.Background(), "inspect", false, approve, emit); err == nil || e.prepared != 0 {
		t.Fatalf("ungrounded call prepared: %d %v", e.prepared, err)
	}
}

func TestGroundingPersistenceFailureBlocksPreparation(t *testing.T) {
	g := &generator{replies: []string{"TOOL read_file", "README.md"}}
	e, j := &executor{}, &journal{failKind: "self_grounding"}
	r := fixture(g, e, j)
	if _, err := r.Run(context.Background(), "inspect", false, approve, emit); err == nil || e.prepared != 0 {
		t.Fatalf("uncommitted grounding executed: %d %v", e.prepared, err)
	}
}

func TestGroundingPersistsAndStaysInContext(t *testing.T) {
	g := &generator{replies: []string{"TOOL read_file", "README.md", "WAIT\nreview"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	if _, err := r.Run(context.Background(), "inspect", false, approve, emit); err != nil {
		t.Fatal(err)
	}
	groundIndex, startIndex := -1, -1
	for i, event := range j.events {
		if event.Grounding != nil {
			groundIndex = i
			if !event.Grounding.Matches(e.calls[0]) {
				t.Fatal("wrong binding")
			}
		}
		if event.Kind == "tool_started" {
			startIndex = i
		}
	}
	if groundIndex < 0 || startIndex <= groundIndex {
		t.Fatal("grounding not durable before execution")
	}
	if len(g.groundingRequests) != 1 {
		t.Fatal("one grounding per call required")
	}
	last := g.requests[len(g.requests)-1]
	found := false
	for _, message := range last.Messages {
		found = found || strings.Contains(message.Content, "Self-grounding for")
	}
	if !found {
		t.Fatal("grounding disappeared from context")
	}
}
