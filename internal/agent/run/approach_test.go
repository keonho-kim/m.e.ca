// Purpose: Keep working choices in history without repeating them in active observations.
// Pattern: Scripted model contract tests.
// Usage: go test ./internal/agent/run -run TestApproach.
// Related: internal/agent/run/work_context.go, internal/agent/run/prepare_execution.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"testing"
)

type notedGenerator struct {
	base  *generator
	notes []string
}

func (g notedGenerator) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	return g.base.Generate(ctx, r, func(e modelapi.Event) error {
		if len(e.Calls) > 0 {
			e.Delta = g.notes[len(g.base.requests)-1]
		}
		return emit(e)
	})
}

func TestApproachRemainsInHistoryWithoutDuplicateDecisionFields(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "a"}), nativeCall("inspect_fixture", map[string]any{"path": "b"}), "Inspected a and b"}}
	r := fixture(g, &executor{}, &journal{})
	r.Generator = notedGenerator{base: g, notes: []string{"Keep the implementation limited to users and posts.", "Inspect b before implementing the database connection."}}
	if _, err := r.Run(context.Background(), "Implement the basic service", approve, emit); err != nil {
		t.Fatal(err)
	}
	request := g.requests[2]
	text := request.Messages[len(request.Messages)-1].Content
	if strings.Contains(text, "proposed_approach") {
		t.Fatal("active observations duplicate assistant proposals")
	}
	var history strings.Builder
	for _, message := range request.Messages {
		history.WriteString(message.Content)
	}
	for _, decision := range []string{"users and posts", "database connection"} {
		if strings.Count(history.String(), decision) != 1 {
			t.Fatal("working choice was lost or duplicated: " + decision)
		}

	}
}
