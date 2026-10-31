// Purpose: Keep foreground and background actual token snapshots independent during overlap.
// Pattern: Terminal accounting regression test.
// Usage: go test ./internal/tui/session -run TestLATMemoryToken.
// Related: internal/tui/session/tokens.go, internal/agent/run/lat_memory_usage.go.
package session

import (
	"testing"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestLATMemoryTokenSnapshotsDoNotReplaceForeground(t *testing.T) {
	a := &app{}
	a.receiveRuntime(&runrecord.Event{Kind: "usage_progress", Usage: &modelapi.TokenUsage{Input: 100, Output: 10}})
	background := &runrecord.Generation{ID: "generation_memory", Role: inference.LATCompaction, Attempt: 1}
	a.receiveRuntime(&runrecord.Event{Kind: "usage_progress", Usage: &modelapi.TokenUsage{Input: 20, Output: 2}, Generation: background})
	if got := a.displayTokens(); got.Input != 120 || got.Output != 12 {
		t.Fatal("overlap lost a measured snapshot", got)
	}
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Input: 20, Output: 3}, Generation: background})
	if got := a.displayTokens(); got.Input != 120 || got.Output != 13 || a.tokenCurrent.Input != 100 {
		t.Fatal("background final reset foreground usage", got)
	}
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Input: 100, Output: 11}})
	if got := a.displayTokens(); got.Input != 120 || got.Output != 14 {
		t.Fatal("completion duplicated or omitted usage", got)
	}
}
