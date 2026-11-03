// Purpose: Verify archived public speech omits empty labels and keeps source bodies separate from attribution.
// Pattern: Pure history projection regression tests.
// Usage: go test ./internal/tui/transcript -run TestTeam.
// Related: internal/tui/transcript/team.go, internal/tui/transcript/history.go.
package transcript

import (
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestTeamHistoryPreservesOnlySubstantiveSpeech(t *testing.T) {
	f := Formatter{Redact: func(text string) string { return text }}
	var records []history.Record
	for _, event := range []runrecord.Event{
		{Kind: "team_progress", Agent: "Hana", Text: "\n\n"},
		{Kind: "team_message", Agent: "Ren", Text: " \t\n"},
		{Kind: "team_progress", Agent: "Hana", Text: "\n\nA-public\n"},
		{Kind: "team_progress", Agent: "Ren", Text: "B-public", Final: true},
	} {
		records = append(records, history.Record{Kind: "event", Event: event})
	}
	blocks := f.History(records)
	if len(blocks) != 4 || !blocks[0].KeepWithNext || !blocks[2].KeepWithNext {
		t.Fatalf("public speech lost attribution or retained blank labels: %+v", blocks)
	}
	if blocks[1].Text != "\n\nA-public\n" || blocks[3].Text != "B-public" {
		t.Fatal("public body was changed or assigned to another message")
	}
}
