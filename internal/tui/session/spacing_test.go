// Purpose: Verify empty streamed and completed model output does not move terminal content.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run TestWhitespace.
// Related: internal/tui/session/render_chat.go, internal/tui/session/runtime_events.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/tui/transcript"
	"strings"
	"testing"
)

func TestWhitespaceModelOutputDoesNotMoveScreen(t *testing.T) {
	a := fixture(t)
	a.output.Append(transcript.User("create file"), false)
	snapshot := func() string {
		a.render()
		var result strings.Builder
		for row := 0; row < 24; row++ {
			result.WriteString(screenLine(a.screen, row, 80))
			result.WriteByte('\n')
		}
		return result.String()
	}
	initial := snapshot()
	raw := " \n\n\t\n"
	a.receiveRuntime(&run.Event{Kind: "stream", Text: raw, Thinking: raw})
	if a.stream != raw || a.thinking != raw {
		t.Fatal("display normalization changed model snapshots")
	}
	if current := snapshot(); current != initial {
		t.Fatal("whitespace stream moved terminal content")
	}
	for _, kind := range []string{"thinking", "progress", "response"} {
		a.receiveRuntime(&run.Event{Kind: kind, Text: raw, Thinking: raw})
		if current := snapshot(); current != initial {
			t.Fatalf("%s appended blank transcript rows", kind)
		}
	}
	if len(a.output.Blocks) != 1 {
		t.Fatal("whitespace persisted into viewport blocks")
	}
}
