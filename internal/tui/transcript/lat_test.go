// Purpose: Verify live and restored LAT outcomes share localized advisory labels.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/transcript -run TestLAT.
// Related: internal/tui/transcript/lat.go, internal/tui/transcript/events.go.
package transcript

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestLATOutcomesAreLocalizedAndKeepFeedback(t *testing.T) {
	for _, language := range []locale.Language{locale.Korean, locale.English} {
		f := Formatter{Language: language, Redact: func(s string) string { return s }}
		for _, kind := range []string{"lat_clean", "lat_feedback", "lat_unavailable", "lat_discarded"} {
			text := ""
			if kind == "lat_feedback" {
				text = "- api.py: add the requested endpoint."
			}
			blocks := f.Event(runrecord.Event{Kind: kind, Text: text})
			if len(blocks) != 1 || !blocks[0].Markdown || !strings.HasPrefix(blocks[0].Text, "LAT:") || !strings.Contains(blocks[0].Text, text) {
				t.Fatalf("invalid review display: %s %+v", kind, blocks)
			}
			if kind == "lat_unavailable" && strings.Contains(blocks[0].Text, "No actionable findings") {
				t.Fatal("failed review shown as clean")
			}
		}
		if blocks := f.Event(runrecord.Event{Kind: "lat_started"}); len(blocks) != 0 {
			t.Fatal("transient review status leaked into history")
		}
	}
}
