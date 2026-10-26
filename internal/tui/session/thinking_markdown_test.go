// Purpose: Verify streamed, completed, interrupted and restored Thinking shares muted Markdown rendering.
// Pattern: Virtual terminal and temporary-history integration tests.
// Usage: go test ./internal/tui/session -run TestThinkingMarkdown.
// Related: internal/tui/session/chat_projection.go, internal/tui/session/runtime_events.go, internal/tui/transcript/events.go.
package session

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/agent/run"
)

const thinkingMarkdown = "# Review\n\n- **inspect** `main.go`\n\n```text\n**literal code**\n```\n\n3~5, ~~keep~~ 😀"

func TestThinkingMarkdownStreaming(t *testing.T) {
	a := fixture(t)
	for _, source := range []string{"# Review", thinkingMarkdown} {
		a.receive(notice{Event: &run.Event{Kind: "stream", Thinking: source}})
		a.render()
		if line := screenLine(a.screen, 1, 80); !strings.Contains(line, "Review") || strings.Contains(line, "# Review") {
			t.Fatalf("Thinking snapshot remained literal: %q", line)
		}
		if a.thinking != source {
			t.Fatal("rendering changed the raw Thinking snapshot")
		}
	}
	checkThinkingMarkdown(t, a)
}

func TestThinkingMarkdownCompletionAndInterruption(t *testing.T) {
	for _, route := range []string{"thinking", "response", "interrupted"} {
		t.Run(route, func(t *testing.T) {
			a := fixture(t)
			if route == "interrupted" {
				a.worker = workerAgentStopping
				a.receive(notice{Event: &run.Event{Kind: "stream", Thinking: thinkingMarkdown}})
				a.receive(notice{Done: true, Err: context.Canceled})
			} else {
				a.receive(notice{Event: &run.Event{Kind: route, Thinking: thinkingMarkdown}})
			}
			if len(a.output.Blocks) != 1 || a.output.Blocks[0].Text != thinkingMarkdown || !a.output.Blocks[0].Thinking || !a.output.Blocks[0].Markdown {
				t.Fatal("Thinking lost its raw source or Markdown presentation")
			}
			checkThinkingMarkdown(t, a)
		})
	}
}

func TestThinkingMarkdownHistory(t *testing.T) {
	a := fixture(t)
	if err := a.opts.Store.RecordEvent(a.ctx, a.opts.Session.ID, run.Event{Kind: "thinking", Thinking: thinkingMarkdown}); err != nil {
		t.Fatal(err)
	}
	if err := a.loadHistory(); err != nil {
		t.Fatal(err)
	}
	if len(a.output.Blocks) != 1 || a.output.Blocks[0].Text != thinkingMarkdown || !a.output.Blocks[0].Thinking || !a.output.Blocks[0].Markdown {
		t.Fatal("history did not restore the original Thinking as Markdown")
	}
	checkThinkingMarkdown(t, a)
}

func checkThinkingMarkdown(t *testing.T, a *app) {
	t.Helper()
	a.render()
	var body strings.Builder
	for y := 1; y < a.inputRegion.y; y++ {
		body.WriteString(screenLine(a.screen, y, 80))
		body.WriteByte('\n')
	}
	for _, want := range []string{"Review", "• inspect main.go", "**literal code**", "3~5, ~~keep~~ 😀"} {
		if !strings.Contains(body.String(), want) {
			t.Fatalf("Thinking Markdown is missing %q: %s", want, body.String())
		}
	}
	if strings.Contains(body.String(), "# Review") || strings.Contains(body.String(), "`main.go`") {
		t.Fatal("Thinking markup was shown literally")
	}
	_, style, _ := a.screen.Get(0, 1)
	if want := tcell.AttrBold | tcell.AttrDim; style.GetAttributes()&want != want || style.GetBackground() != tcell.ColorDefault {
		t.Fatal("Thinking heading lost its bold, dim or default-background style")
	}
}
