// Purpose: Verify live completion and saved speech share masking, spacing and final attribution.
// Pattern: Deterministic presentation contract tests.
// Usage: go test ./internal/tui/chat/team -run TestTeamCompletion.
// Related: internal/tui/chat/team/events.go, internal/tui/transcript/team.go, internal/tui/chat/team/message.go.
package team

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestTeamCompletionKeepsOwnPairAndMatchesSavedSpeech(t *testing.T) {
	f := formatter()
	f.Redact = func(text string) string { return strings.ReplaceAll(text, "secret", "[hidden]") }
	var state State
	output := viewport.View{Follow: true}
	state.Apply(runrecord.Event{Kind: "stream", Agent: "Hana", Text: "draft"}, f, &output)
	owned := state.peers["Hana"].message
	event := runrecord.Event{Kind: "team_progress", Agent: "Hana", Text: "**result** secret", Final: true}
	state.Apply(event, f, &output)
	saved := f.Event(event)
	if len(output.Blocks) != 2 || len(saved) != 2 || output.Blocks[0].ID != owned.header || output.Blocks[1].ID != owned.body {
		t.Fatal("completion duplicated the existing response or lost its source identity")
	}
	for i, block := range output.Blocks {
		if block.Text != saved[i].Text || block.SpaceAfter != saved[i].SpaceAfter || strings.Contains(block.Text, "secret") {
			t.Fatalf("live and saved public speech differ: %+v", block)
		}
	}
	if output.Blocks[1].Streaming || state.peers["Hana"].message != (messageDisplay{}) {
		t.Fatal("completed speech retained active stream ownership")
	}
}
