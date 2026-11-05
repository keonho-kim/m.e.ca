// Purpose: Discard failed generation drafts while preserving other peers and committed replies.
// Pattern: Deterministic generation-boundary regressions.
// Usage: go test ./internal/tui/chat/team -run TestRetry.
// Related: internal/tui/chat/team/events.go, internal/tui/chat/team/message.go.
package team

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestRetryDiscardsOnlyItsUncommittedDraft(t *testing.T) {
	var state State
	output := viewport.View{Follow: true, Collapsed: true}
	for _, event := range []runrecord.Event{
		{Kind: "stream", Agent: "Hana", Text: "Committed response"},
		{Kind: "team_progress", Agent: "Hana", Text: "Committed response"},
		{Kind: "generation_started", Agent: "Hana"},
		{Kind: "stream", Agent: "Hana", Text: "DISCARDED_DRAFT"},
		{Kind: "stream", Agent: "Ren", Text: "Other peer draft"},
		{Kind: "retry", Agent: "Hana", Text: "Provider retry 2/3"},
		{Kind: "generation_started", Agent: "Hana"},
		{Kind: "stream", Agent: "Hana", Text: "Accepted response"},
		{Kind: "team_progress", Agent: "Hana", Text: "Accepted response"},
	} {
		state.Apply(event, formatter(), &output)
	}
	var text strings.Builder
	for _, block := range output.Blocks {
		text.WriteString(block.Text)
	}
	if strings.Contains(text.String(), "DISCARDED_DRAFT") {
		t.Fatal("failed generation draft remained in transcript")
	}
	for _, want := range []string{"Committed response", "Other peer draft", "Accepted response"} {
		if strings.Count(text.String(), want) != 1 {
			t.Fatalf("retry changed another response: %q", text.String())
		}
	}
}
