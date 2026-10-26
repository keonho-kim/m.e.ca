// Purpose: Verify intermediate updates render once with matching live and restored spacing.
// Pattern: Virtual terminal presentation test.
// Usage: go test ./internal/tui/session -run TestProgressUpdate.
// Related: internal/tui/session/runtime_events.go, internal/tui/transcript/events.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"strings"
	"testing"
)

func TestProgressUpdateLiveMatchesRestored(t *testing.T) {
	a := fixture(t)
	event := runrecord.Event{Kind: "progress", Text: "Reviewing the existing file."}
	a.receive(notice{Event: &event})
	formatter := transcript.Formatter{Redact: a.opts.Vault.Redact}
	restored := formatter.Event(event)
	want := transcript.Reply(event.Text)
	if len(a.output.Blocks) != 1 || a.output.Blocks[0].Text != want || len(restored) != 1 || restored[0].Text != want {
		t.Fatalf("live=%+v restored=%+v", a.output.Blocks, restored)
	}
	if blocks := formatter.Message(modelapi.Message{Role: "system", Content: event.Text}); len(blocks) != 0 {
		t.Fatal("context copy duplicated in transcript")
	}
	a.render()
	var screen strings.Builder
	for row := 0; row < 24; row++ {
		screen.WriteString(screenLine(a.screen, row, 80))
	}
	if !strings.Contains(screen.String(), event.Text) {
		t.Fatal("progress missing from terminal")
	}
}
