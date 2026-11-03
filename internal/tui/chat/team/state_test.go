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
	"time"
)

func formatter() transcript.Formatter {
	return transcript.Formatter{Redact: func(text string) string { return text }}
}
func TestStateAggregatesOnlyReportedUsage(t *testing.T) {
	var state State
	state.Begin()
	output := viewport.View{Follow: true}
	f := formatter()
	for _, id := range []string{"Hana", "Ren"} {
		state.Apply(runrecord.Event{Kind: "team_notice", Agent: id, Text: "joined"}, f, &output)
	}
	// Repeated membership notices must not inflate the cached active count.
	state.Apply(runrecord.Event{Kind: "team_notice", Agent: "Hana", Text: "joined"}, f, &output)
	if state.Count() != 2 {
		t.Fatal(state.Count())
	}
	measured := 2 * time.Second
	state.Apply(runrecord.Event{Kind: "usage", Agent: "Hana", Usage: &modelapi.TokenUsage{Input: 100, Output: 20}, Generation: &runrecord.Generation{MeasuredMillis: 1000}}, f, &output)
	state.Apply(runrecord.Event{Kind: "usage_progress", Agent: "Ren", Usage: &modelapi.TokenUsage{Input: 200, Output: 40}, MeasuredDuration: measured}, f, &output)
	usage, tps := state.CurrentUsage()
	if usage.Input != 200 || usage.Output != 40 || tps == nil || *tps != 20 {
		t.Fatalf("usage=%+v tps=%v", usage, tps)
	}
	state.Apply(runrecord.Event{Kind: "compaction_started", Agent: "Hana"}, f, &output)
	state.Apply(runrecord.Event{Kind: "compaction_started", Agent: "Ren"}, f, &output)
	state.Apply(runrecord.Event{Kind: "compaction_finished", Agent: "Hana"}, f, &output)
	if !state.Compressing() {
		t.Fatal("one peer still compacts")
	}
	state.Apply(runrecord.Event{Kind: "team_notice", Agent: "Ren", Text: "left"}, f, &output)
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
		{Kind: "stream", Agent: "Hana", Text: "first", Thinking: "private"},
		{Kind: "stream", Agent: "Ren", Text: "second"},
		{Kind: "stream", Agent: "Hana", Text: "first completed"},
		{Kind: "team_progress", Agent: "Hana", Text: "first completed"},
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
	if state.peers != nil || state.tools != nil || state.measured != (runrecord.Throughput{}) {
		t.Fatal("closed projection retained per-request data")
	}
	for _, block := range output.Blocks {
		if block.Streaming {
			t.Fatal("finished projection retained active stream")
		}
	}
}
