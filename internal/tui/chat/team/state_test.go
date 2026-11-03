// Purpose: Verify independent display streams, aggregate usage and complete projection cleanup without a terminal.
// Pattern: Pure feature-state behavior tests.
// Usage: go test ./internal/tui/chat/team.
// Related: internal/tui/chat/team/state.go, internal/tui/chat/team/events.go.
package team

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
	"strings"
	"testing"
)

func formatter() transcript.Formatter {
	return transcript.Formatter{Redact: func(text string) string { return text }}
}
func TestStateAggregatesOnlyReportedUsage(t *testing.T) {
	var state State
	state.Begin()
	output := viewport.View{Follow: true}
	f := formatter()
	for _, id := range []string{"agent-a", "agent-b"} {
		state.Apply(runrecord.Event{Kind: "team_notice", Agent: id, Text: "joined"}, f, &output)
	}
	// Repeated membership notices must not inflate the cached active count.
	state.Apply(runrecord.Event{Kind: "team_notice", Agent: "agent-a", Text: "joined"}, f, &output)
	if state.Count() != 2 {
		t.Fatal(state.Count())
	}
	rate := 20.0
	state.Apply(runrecord.Event{Kind: "usage", Agent: "agent-a", Usage: &modelapi.TokenUsage{Input: 100, Output: 20}, Generation: &runrecord.Generation{MeasuredMillis: 1000}}, f, &output)
	state.Apply(runrecord.Event{Kind: "usage_progress", Agent: "agent-b", Usage: &modelapi.TokenUsage{Input: 200, Output: 40}, TokensPerSecond: &rate}, f, &output)
	usage, tps := state.CurrentUsage()
	if usage.Input != 200 || usage.Output != 40 || tps == nil || *tps != 20 {
		t.Fatalf("usage=%+v tps=%v", usage, tps)
	}
	state.Apply(runrecord.Event{Kind: "compaction_started", Agent: "agent-a"}, f, &output)
	state.Apply(runrecord.Event{Kind: "compaction_started", Agent: "agent-b"}, f, &output)
	state.Apply(runrecord.Event{Kind: "compaction_finished", Agent: "agent-a"}, f, &output)
	if !state.Compressing() {
		t.Fatal("one peer still compacts")
	}
	state.Apply(runrecord.Event{Kind: "team_notice", Agent: "agent-b", Text: "left"}, f, &output)
	if state.Compressing() || state.Count() != 1 {
		t.Fatal("closed peer retained active compression")
	}
}
func TestStatePreservesStreamsAndReleasesOwnedReferences(t *testing.T) {
	var state State
	state.Begin()
	output := viewport.View{Follow: true}
	f := formatter()
	for _, event := range []runrecord.Event{
		{Kind: "stream", Agent: "agent-a", Text: "first", Thinking: "private"},
		{Kind: "stream", Agent: "agent-b", Text: "second"},
		{Kind: "stream", Agent: "agent-a", Text: "first completed"},
		{Kind: "team_progress", Agent: "agent-a", Text: "first completed"},
	} {
		state.Apply(event, f, &output)
	}
	var text strings.Builder
	for _, block := range output.Blocks {
		text.WriteString(block.Text)
	}
	if strings.Count(text.String(), "first completed") != 1 || strings.Contains(text.String(), "private") {
		t.Fatal(text.String())
	}
	state.Finish(&output)
	if state.peers != nil || state.tools != nil || state.totals.Received {
		t.Fatal("closed projection retained per-request data")
	}
	for _, block := range output.Blocks {
		if block.Streaming {
			t.Fatal("finished projection retained active stream")
		}
	}
}
