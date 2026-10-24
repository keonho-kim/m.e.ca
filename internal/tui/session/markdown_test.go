// Purpose: Verify live and restored agent Markdown renders while tool and user text stays literal.
// Pattern: Simulated terminal workflow tests.
// Usage: go test ./internal/tui/session -run TestMarkdown.
// Related: internal/tui/session/runtime_events.go, internal/tui/session/render_chat.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/run"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"strings"
	"testing"
)

func TestMarkdownLiveAndRestored(t *testing.T) {
	a := fixture(t)
	a.receive(notice{Event: &run.Event{Kind: "stream", Text: "# Heading\n\n**strong** 😀"}})
	a.render()
	first := screenLine(a.screen, 1, 80)
	if !strings.Contains(first, "Heading") || strings.Contains(first, "# Heading") {
		t.Fatalf("stream not rendered: %q", first)
	}
	_, style, _ := a.screen.Get(0, 1)
	if style.GetAttributes()&tcell.AttrBold == 0 {
		t.Fatal("heading is not bold")
	}
	a.receive(notice{Event: &run.Event{Kind: "response", Text: "# Heading\n\n**strong** 😀"}})
	formatter := transcript.Formatter{Redact: func(s string) string { return s }}
	saved := formatter.Message(modelapi.Message{Role: "assistant", Content: "# Heading\n\n**strong** 😀"})
	if len(saved) != 1 || !saved[0].Markdown || !a.output.Blocks[0].Markdown {
		t.Fatal("live and restored markup mismatch")
	}
	a.screen.SetSize(12, 24)
	a.render()
	for y := 1; y < 20; y++ {
		for x := 0; x < 12; x++ {
			_, style, _ := a.screen.Get(x, y)
			if style.GetBackground() != tcell.ColorDefault {
				t.Fatal("Markdown introduced background colour")
			}
		}
	}
}
func TestMarkdownUserRemainsLiteral(t *testing.T) {
	formatter := transcript.Formatter{Redact: func(s string) string { return s }}
	block := formatter.Message(modelapi.Message{Role: "user", Content: "**literal**"})[0]
	if block.Markdown || !strings.Contains(block.Text, "**literal**") {
		t.Fatal("user text interpreted as agent Markdown")
	}
}

func TestMarkdownToolOutputRemainsLiteral(t *testing.T) {
	a := fixture(t)
	a.receive(notice{Event: &run.Event{Kind: "tool_succeeded", Call: &toolapi.Call{ID: "read", Name: "execute_command", Arguments: []byte(`{"command":"cat a"}`)}, Text: "**literal tool output**"}})
	if len(a.output.Blocks) != 1 || a.output.Blocks[0].Markdown || !strings.Contains(a.output.Blocks[0].Text, "**literal tool output**") {
		t.Fatal("tool output was interpreted as Markdown")
	}
}
