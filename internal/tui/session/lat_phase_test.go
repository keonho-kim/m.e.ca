// Purpose: Verify preparation phases show localized transient status without claiming a final review result.
// Pattern: Virtual terminal behavior test.
// Usage: go test ./internal/tui/session -run TestLATPreparation.
// Related: internal/tui/session/runtime_events.go, internal/tui/locale/messages.go.
package session

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestLATPreparationStatusIsDistinctFromFinalReview(t *testing.T) {
	for _, language := range []locale.Language{locale.English, locale.Korean} {
		a := screenFixture(t)
		a.language, a.worker = language, workerAgent
		for _, phase := range []struct{ name, label string }{{"plan", "LAT: Preparing plan…"}, {"critique", "LAT: Reviewing plan…"}, {"refine", "LAT: Refining plan…"}} {
			a.receiveRuntime(&runrecord.Event{Kind: "lat_preparing", Text: phase.name})
			want := language.Text(phase.label)
			if a.status != want || len(a.output.Blocks) != 0 {
				t.Fatalf("preparation became transcript content: status=%q blocks=%+v", a.status, a.output.Blocks)
			}
			a.render()
			var screen strings.Builder
			for row := range 24 {
				screen.WriteString(screenLine(a.screen, row, 80))
			}
			if !strings.Contains(screen.String(), want) {
				t.Fatalf("phase status missing from terminal: %q", want)
			}
		}
		a.receiveRuntime(&runrecord.Event{Kind: "lat_prepared"})
		if a.status != "" || len(a.output.Blocks) != 0 {
			t.Fatalf("preparation completion claimed a review result: status=%q blocks=%+v", a.status, a.output.Blocks)
		}
		a.receiveRuntime(&runrecord.Event{Kind: "lat_started"})
		if a.status != language.Text("LAT: Reviewing work…") || len(a.output.Blocks) != 0 {
			t.Fatalf("final review changed: status=%q blocks=%+v", a.status, a.output.Blocks)
		}
	}
}
