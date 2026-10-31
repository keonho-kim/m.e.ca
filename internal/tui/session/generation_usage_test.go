// Purpose: Verify sequential generation snapshots commit once and compaction releases its display.
// Pattern: Event-loop projection tests.
// Usage: go test ./internal/tui/session -run TestGeneration.
// Related: internal/tui/session/tokens.go, internal/tui/session/activity.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"testing"
)

func TestGenerationTokenSnapshotsAndFinals(t *testing.T) {
	a := screenFixture(t)
	rate := 5.0
	update := func(kind string, input, output int64) {
		a.receiveRuntime(&runrecord.Event{Kind: kind, Usage: &modelapi.TokenUsage{Input: input, Output: output}, TokensPerSecond: &rate})
	}
	update("usage_progress", 100, 10)
	update("usage_progress", 100, 15)
	if got := a.displayTokens(); got.Input != 100 || got.Output != 15 || *a.tokensPerSecond != 5 {
		t.Fatalf("snapshots duplicated: %+v", got)
	}
	update("usage", 100, 15)
	a.receiveRuntime(&runrecord.Event{Kind: "generation_started"})
	if a.tokensPerSecond != nil {
		t.Fatal("new generation inherited throughput")
	}
	update("usage_progress", 200, 25)
	update("usage", 200, 25)
	update("usage", 50, 5)
	if got := a.displayTokens(); got.Input != 350 || got.Output != 45 || a.tokenCurrent != (modelapi.TokenUsage{}) {
		t.Fatalf("final usage duplicated or lost: %+v", got)
	}
}
func TestGenerationCompactionDisplayStops(t *testing.T) {
	a := screenFixture(t)
	defer a.stopCompression()
	a.receiveRuntime(&runrecord.Event{Kind: "compaction_started"})
	if !a.activity().Compressing {
		t.Fatal("compression did not start")
	}
	a.receiveRuntime(&runrecord.Event{Kind: "compaction_finished"})
	if a.activity().Compressing {
		t.Fatal("compression display retained after completion")
	}
}
