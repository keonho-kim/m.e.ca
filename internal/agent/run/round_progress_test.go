// Purpose: Verify one model action group receives one progress judgement, including skipped siblings.
// Pattern: Scripted runtime regression tests.
// Usage: go test ./internal/agent/run -run TestRoundProgress.
// Related: internal/agent/run/round_progress.go, internal/agent/run/batch.go.
package run

import (
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"testing"
)

type repeatedReadGroup struct {
	turns int
}

func (g *repeatedReadGroup) Generate(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	g.turns++
	if g.turns == 3 {
		return emit(modelapi.Event{Delta: "The files were inspected; proceeding with the result.", Done: true})
	}
	var calls []toolapi.Call
	for i := 0; i < 4; i++ {
		calls = append(calls, batchCall(fmt.Sprintf("read-%d-%d", g.turns, i), "inspect_fixture", map[string]any{"path": fmt.Sprintf("file-%d", i)}))
	}
	return emit(modelapi.Event{Calls: calls, Done: true})
}
func TestRoundProgressRepeatedReadBatchCountsOnce(t *testing.T) {
	g, j := &repeatedReadGroup{}, &journal{}
	r := fixture(&generator{}, &executor{}, j)
	r.Generator = g
	result, err := r.Run(context.Background(), "inspect files", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("%+v %v", result, err)
	}
	maximum := 0
	for _, event := range j.events {
		if cp := event.Checkpoint; cp != nil {
			maximum = max(maximum, cp.NoProgress)
		}
	}
	if maximum != 1 {
		t.Fatalf("one repeated batch counted as %d stalled rounds", maximum)
	}
}

func TestRoundProgressRejectedWaveAllowsCorrection(t *testing.T) {
	calls := []toolapi.Call{}
	for i := 0; i < 4; i++ {
		calls = append(calls, batchCall(fmt.Sprintf("bad-%d", i), "inspect_fixture", map[string]any{"path": fmt.Sprintf("missing-%d", i)}))
	}
	r, _, _, j, g := realBatchRunner(t, calls)
	result, err := r.Run(context.Background(), "inspect files", approve, emit)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.requests) != 2 {
		t.Fatalf("failed batch ended without the next model decision: %+v", result)
	}
	for _, event := range j.events {
		if cp := event.Checkpoint; cp != nil && cp.NoProgress > 1 {
			t.Fatalf("skipped siblings counted repeatedly: %d", cp.NoProgress)
		}
	}
}
