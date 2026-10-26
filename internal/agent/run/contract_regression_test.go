// Purpose: Verify failed calls, exact final drafts and response streaming obey completion contracts.
// Pattern: Scripted runtime regression tests.
// Usage: go test ./internal/agent/run -run TestContract.
// Related: internal/agent/run/finish.go, internal/agent/run/generate.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestContractInvalidBatchExecutesNothing(t *testing.T) {
	calls := []toolapi.Call{batchCall("bad", "write_file", map[string]any{"path": "a"}), batchCall("read", "inspect_fixture", map[string]any{"path": "a"})}
	r, _, _, j, _ := realBatchRunner(t, calls)
	result, err := r.Run(context.Background(), "Create a", approve, emit)
	if err != nil || !result.HadToolFailure {
		t.Fatalf("failed batch accepted as completion: %+v %v", result, err)
	}
	for _, e := range j.events {
		if e.Kind == "tool_started" {
			t.Fatal("invalid batch executed")
		}
	}
}
func TestContractEntireAnswerIsCommittedWithoutRegeneration(t *testing.T) {
	tail := "The requested file was read. No deployment was performed."
	draft := strings.Repeat("Observed file contents. ", 500) + tail
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "a"}), draft}}
	result, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "Read a", approve, emit)
	if err != nil || result.Text != draft {
		t.Fatalf("final differs: %v", err)
	}
}
func TestContractPostToolResponseStreams(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	r.Generator = &chunkGenerator{chunks: []string{"Hello", " world"}}
	visible := 0
	s := execution{runner: r, limits: DefaultLimits(), usage: &requestBudget{}, system: "rules", checkpoint: runrecord.Checkpoint{Phase: "decision", Work: toolapi.Facts{}}, emit: func(e runrecord.Event) error {
		if e.Kind == "stream" && e.Text != "" {
			visible++
		}
		return nil
	}}
	reply, err := s.askResponse(context.Background())
	if err != nil || reply.Text != "Hello world" || visible != 2 {
		t.Fatalf("stream events=%d error=%v", visible, err)
	}
}

type malformedWorkGenerator struct{ failed bool }

func (g *malformedWorkGenerator) Generate(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	if !g.failed {
		g.failed = true
		return &modelapi.ToolCallError{Detail: "invalid generated arguments"}
	}
	return emit(modelapi.Event{Delta: "Completed the requested file.", Done: true})
}
func TestContractMalformedToolResponseRemainsAFailedObservation(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	r.Generator = &malformedWorkGenerator{}
	result, err := r.Run(context.Background(), "Create a", approve, emit)
	if err != nil || !result.HadToolFailure {
		t.Fatalf("%+v %v", result, err)
	}
}
