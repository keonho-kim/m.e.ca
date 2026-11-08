// Purpose: Reject stale single-agent candidates while preserving completed effects and native replies.
// Pattern: Scripted instruction-boundary regression tests.
// Usage: Runtime tests inject user steering during generation, between waves and during approval.
// Related: internal/agent/run/steering.go, internal/agent/run/execute.go, internal/agent/run/batch.go.
package run

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	"context"
	"strings"
	"testing"

	model "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func TestSingleSteeringDuringGenerationRejectsOldCandidate(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "auth.py", "content": "obsolete"}), "Preserved authentication"}}
	tools, j := &executor{}, &journal{}
	r := fixture(g, tools, j)
	delivered := false
	r.PollSteering = func() []string {
		if len(g.requests) == 1 && !delivered {
			delivered = true
			return []string{"Keep auth.py unchanged", "Continue implementation"}
		}
		return nil
	}
	if _, err := r.Run(context.Background(), "Continue implementation", approve, emit); err != nil {
		t.Fatal(err)
	}
	if tools.prepared != 0 || tools.executed != 0 || len(g.requests) != 2 {
		t.Fatal("same final wording hid the newer restriction", tools.prepared, tools.executed, len(g.requests))
	}
	seen := 0
	for _, message := range g.requests[1].Messages {
		if strings.Contains(message.Content, "Keep auth.py unchanged") {
			seen++
		}
	}
	if seen != 1 {
		t.Fatal("steering was missing or injected twice", seen)
	}
}

func TestSingleSteeringBetweenWavesPreservesTheCompletedSibling(t *testing.T) {
	tools, j := &executor{}, &journal{}
	r := fixture(&generator{}, tools, j)
	requests, delivered := 0, false
	r.Generator = boundaryModel(func(_ context.Context, request model.Request, emit func(model.Event) error) error {
		requests++
		if requests == 1 {
			return emit(model.Event{Done: true, Calls: []tool.Call{
				batchCall("first", "write_file", map[string]any{"path": "first", "content": "needed"}),
				batchCall("second", "write_file", map[string]any{"path": "second", "content": "obsolete"}),
			}})
		}
		found := false
		for _, message := range request.Messages {
			found = found || strings.Contains(message.Content, diagcatalog.InstructionChanged)
		}
		if !found {
			t.Error("the next model decision missed why the remaining call was skipped")
		}
		return emit(model.Event{Delta: "Preserved the second file", Done: true})
	})
	r.PollSteering = func() []string {
		if tools.executed == 1 && !delivered {
			delivered = true
			return []string{"Keep second unchanged"}
		}
		return nil
	}
	if _, err := r.Run(context.Background(), "Create both files", approve, emit); err != nil {
		t.Fatal(err)
	}
	if !delivered || tools.executed != 1 || tools.prepared != 1 || requests != 2 {
		t.Fatal("a completed sibling was replayed or an obsolete sibling executed", tools.executed, tools.prepared, requests)
	}
}

func TestSingleSteeringDuringApprovalRejectsThePreparedCandidate(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file", "content": "obsolete"}), "Preserved existing files"}}
	tools, j := &executor{approval: true}, &journal{}
	r := fixture(g, tools, j)
	var pending []string
	r.PollSteering = func() []string { values := pending; pending = nil; return values }
	approve := func(context.Context, tool.Approval) (bool, error) {
		pending = []string{"Do not modify file"}
		return true, nil
	}
	if _, err := r.Run(context.Background(), "Update file", approve, emit); err != nil {
		t.Fatal(err)
	}
	if tools.prepared != 1 || tools.executed != 0 || len(g.requests) != 2 {
		t.Fatal("approval masked a newer user instruction", tools.prepared, tools.executed, len(g.requests))
	}
}
