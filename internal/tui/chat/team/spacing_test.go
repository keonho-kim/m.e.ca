// Purpose: Verify stable team speech spacing through interleaved streams and completed responses.
// Pattern: Deterministic presentation regression tests.
// Usage: go test ./internal/tui/chat/team -run TestSpeechSpacing.
// Related: internal/tui/chat/team/message.go, internal/tui/transcript/team.go, internal/tui/viewport/block_layout.go.
package team

import (
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestSpeechSpacingSurvivesStreamingCompletionAndReflow(t *testing.T) {
	var state State
	output := viewport.View{Follow: true}
	for _, event := range []runrecord.Event{
		{Kind: "stream", Agent: "Hana", Text: "A"},
		{Kind: "stream", Agent: "Ren", Text: "B-public"},
		{Kind: "stream", Agent: "Hana", Text: "A-public\n\n\n\n"},
		{Kind: "team_progress", Agent: "Hana", Text: "A-public"},
		{Kind: "team_progress", Agent: "Ren", Text: "B-public\n"},
	} {
		state.Apply(event, formatter(), &output)
		if len(output.Blocks) != 4 {
			continue
		}
		for _, width := range []int{12, 80} {
			var layout viewport.Transcript
			layout.Prepare(width, &output, nil)
			rows := layout.Window(0, layout.Count())
			if len(rows) != 10 || rows[0].Text != "Hana" || rows[5].Text != "Ren" {
				t.Fatalf("%s width %d: spacing changed: %+v", event.Kind, width, rows)
			}
			for _, row := range []int{1, 3, 4, 6, 8, 9} {
				if rows[row].Text != "" {
					t.Fatalf("%s width %d: row %d is not blank: %q", event.Kind, width, row, rows[row].Text)
				}
			}
			if rows[7].Text != "B-public" {
				t.Fatalf("Ren's body lost attribution: %+v", rows)
			}
		}
	}
}
