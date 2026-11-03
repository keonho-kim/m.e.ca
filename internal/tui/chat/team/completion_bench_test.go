// Purpose: Measure completion of a visible team response without unrelated history growth.
// Pattern: Native event projection allocation benchmark.
// Usage: go test ./internal/tui/chat/team -run '^$' -bench BenchmarkTeamCompletionExistingPair -benchmem.
// Related: internal/tui/chat/team/events.go, internal/tui/chat/team/message.go, internal/tui/transcript/team.go.
package team

import (
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func BenchmarkTeamCompletionExistingPair(b *testing.B) {
	var state State
	output := viewport.View{Follow: true}
	f := formatter()
	event := runrecord.Event{Kind: "team_progress", Agent: "Hana", Text: "Completed contribution."}
	state.Apply(runrecord.Event{Kind: "stream", Agent: event.Agent, Text: event.Text}, f, &output)
	owned := state.peers[event.Agent].message
	b.ReportAllocs()
	for b.Loop() {
		// Reuse one existing response pair to isolate completion projection.
		state.peers[event.Agent].message = owned
		state.Apply(event, f, &output)
	}
}
