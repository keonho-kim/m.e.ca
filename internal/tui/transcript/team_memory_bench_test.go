// Purpose: Measure source sharing and compact tool projection without retaining discarded detail copies.
// Pattern: Presentation allocation benchmark.
// Usage: go test ./internal/tui/transcript -run '^$' -bench 'Benchmark(TeamBody|ToolProjection)' -benchmem.
// Related: internal/tui/transcript/team.go, internal/tui/transcript/events.go.
package transcript

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func BenchmarkTeamBodyProjection(b *testing.B) {
	event := runrecord.Event{Kind: "team_progress", Agent: "agent-a", Text: strings.Repeat("response\n", 128*1024)}
	formatter := Formatter{Redact: func(s string) string { return s }}
	b.ReportAllocs()
	for b.Loop() {
		blocks := formatter.Event(event)
		if len(blocks) != 2 || blocks[1].Text != event.Text {
			b.Fatal("body projection changed source")
		}
	}
}
func BenchmarkToolProjection(b *testing.B) {
	formatter := Formatter{Redact: func(s string) string { return s }}
	for _, compact := range []bool{false, true} {
		name := "expanded"
		if compact {
			name = "compact"
		}
		b.Run(name, func(b *testing.B) {
			event := runrecord.Event{Kind: "tool_succeeded", Call: &toolapi.Call{Name: "edit_file", Arguments: []byte(`{"path":"app.go"}`)}, Output: []byte(`{"changed":true}`), ChangePreview: "--- app.go\n+++ app.go\n@@ -1 +1,10000 @@\n-old\n" + strings.Repeat("+new code\n", 10000)}
			if compact {
				event.Reference = &runrecord.EventReference{Session: "source", Sequence: 1}
			}
			b.ReportAllocs()
			for b.Loop() {
				if len(formatter.Event(event)) != 1 {
					b.Fatal("tool projection missing")
				}
			}
		})
	}
}
