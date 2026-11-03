// Purpose: Verify archived public speech omits empty labels and keeps source bodies separate from attribution.
// Pattern: Pure history projection regression tests.
// Usage: go test ./internal/tui/transcript -run TestTeam.
// Related: internal/tui/transcript/team.go, internal/tui/transcript/history.go.
package transcript

import (
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/history"
	"github.com/keonho-kim/meca/internal/tui/viewport"
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

func TestTeamHistorySpacesNamesAndSpeechWithoutChangingSource(t *testing.T) {
	f := Formatter{Redact: func(text string) string { return text }}
	for _, width := range []int{12, 80} {
		var output viewport.View
		for _, event := range []runrecord.Event{
			{Kind: "team_progress", Agent: "Hana", Text: "\n\nA-public\n\n\n\n"},
			{Kind: "team_message", Agent: "Ren", Text: "B-public"},
		} {
			for _, block := range f.Event(event) {
				output.AppendBlock(block)
			}
		}
		if output.Blocks[1].Text != "\n\nA-public\n\n\n\n" {
			t.Fatal("spacing changed the saved source")
		}
		var layout viewport.Transcript
		layout.Prepare(width, &output, nil)
		rows := layout.Window(0, layout.Count())
		want := []string{"Hana", "", "A-public", "", "", "Ren", "", "B-public", "", ""}
		if len(rows) != len(want) {
			t.Fatalf("width %d: got %d rows, want %d: %+v", width, len(rows), len(want), rows)
		}
		for i, text := range want {
			if rows[i].Text != text {
				t.Fatalf("width %d row %d: %q, want %q", width, i, rows[i].Text, text)
			}
		}
	}
}

func TestTeamToolSpacingKeepsNameAndBodySeparated(t *testing.T) {
	f := Formatter{Redact: func(text string) string { return text }}
	for _, event := range []runrecord.Event{
		{Kind: "tool_succeeded", Agent: "Hana", Call: &toolapi.Call{Name: "write_file", Arguments: []byte(`{"path":"new.txt","content":"hello"}`)}, Text: "created"},
		{Kind: "tool_failed", Agent: "Hana", Call: &toolapi.Call{Name: "execute_command", Arguments: []byte(`{"command":"false"}`)}, Text: "failed"},
	} {
		var output viewport.View
		for _, block := range f.Event(event) {
			output.AppendBlock(block)
		}
		var layout viewport.Transcript
		layout.Prepare(80, &output, nil)
		rows := layout.Window(0, layout.Count())
		if len(rows) < 5 || rows[0].Text != "Hana" || rows[1].Text != "" || rows[2].Text == "" || rows[len(rows)-1].Text != "" || rows[len(rows)-2].Text != "" {
			t.Fatalf("%s: tool spacing changed: %+v", event.Kind, rows)
		}
	}
}
