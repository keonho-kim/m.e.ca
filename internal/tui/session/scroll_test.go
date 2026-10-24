// Purpose: Verify scrolling uses the displayed stream and preserves follow state through compaction.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run TestTranscriptScroll.
// Related: internal/tui/session/history.go, internal/tui/session/render_chat.go, internal/tui/session/runtime_events.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func TestTranscriptScrollIncludesLiveThinkingAndAnswer(t *testing.T) {
	a := fixture(t)
	a.historyEnd = true
	a.output.Append("earlier conversation", false)
	a.receiveRuntime(&run.Event{Kind: "stream", Thinking: strings.Repeat("thinking\n", 40), Text: strings.Repeat("answer\n", 40) + "latest"})
	a.render()
	height := a.chatHeight(80, a.bodyHeight(24))
	transcript := a.chatLayout(view.TranscriptWidth(80))
	before := transcript.Start(&a.output, height)
	a.scroll(-3, 80, height)
	if a.output.Follow || transcript.Start(&a.output, height) != before-3 {
		t.Fatalf("live output ignored: before=%d after=%d follow=%t", before, transcript.Start(&a.output, height), a.output.Follow)
	}
	a.scroll(3, 80, height)
	if !a.output.Follow || transcript.Start(&a.output, height) != before {
		t.Fatal("did not resume following the actual displayed bottom")
	}
}

func TestTranscriptScrollCompactionKeepsFollowAndManualPosition(t *testing.T) {
	for _, paused := range []bool{false, true} {
		t.Run(map[bool]string{false: "following", true: "reading"}[paused], func(t *testing.T) {
			a := fixture(t)
			a.historyEnd = true
			defer a.stopCompression()
			a.output.Append(strings.Repeat("earlier\n", 50), false)
			a.receiveRuntime(&run.Event{Kind: "stream", Thinking: strings.Repeat("thinking\n", 40)})
			a.render()
			height := a.chatHeight(80, a.bodyHeight(24))
			if paused {
				a.scroll(-4, 80, height)
			}
			before := a.chatLayout(view.TranscriptWidth(80)).Start(&a.output, height)
			a.receiveRuntime(&run.Event{Kind: "thinking", Thinking: a.thinking})
			a.receiveRuntime(&run.Event{Kind: "compaction_started"})
			a.receiveRuntime(&run.Event{Kind: "compaction_finished"})
			a.receiveRuntime(&run.Event{Kind: "stream", Text: strings.Repeat("continued\n", 30) + "latest answer"})
			a.render()
			if a.output.Follow == paused {
				t.Fatal("compaction changed follow preference")
			}
			if paused && a.chatLayout(view.TranscriptWidth(80)).Start(&a.output, height) != before {
				t.Fatal("thinking commit or compaction moved the reading position")
			}
			if !paused && !strings.Contains(screenLine(a.screen, height-1, 79)+screenLine(a.screen, height, 79), "latest answer") {
				t.Fatal("post-compaction answer is not visible at the bottom")
			}
		})
	}
}

func TestTranscriptScrollbarClickAndDrag(t *testing.T) {
	a := fixture(t)
	a.historyEnd = true
	a.output.Append(strings.Repeat("history\n", 100)+"latest", false)
	a.render()
	height := a.chatHeight(80, a.bodyHeight(24))
	if !strings.Contains(screenLine(a.screen, 1, 80), "│") {
		t.Fatal("scrollbar track missing")
	}
	a.screen.SetSize(240, 24)
	a.render()
	if strings.Contains(footerKeyText(a), "마우스") || strings.Contains(footerKeyText(a), "드래그") {
		t.Fatal("mouse instructions crowded the keyboard hints")
	}
	a.screen.SetSize(80, 24)
	a.render()
	a.mouse(tcell.NewEventMouse(79, 1, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(79, 1, tcell.ButtonNone, 0))
	a.render()
	if a.output.Follow || a.textSelection.Active() {
		t.Fatal("track click did not scroll or started text selection")
	}
	thumb := -1
	for y := 1; y <= height; y++ {
		if strings.HasSuffix(screenLine(a.screen, y, 80), "┃") {
			thumb = y
			break
		}
	}
	if thumb < 0 {
		t.Fatal("scrollbar thumb missing")
	}
	a.mouse(tcell.NewEventMouse(79, thumb, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(70, height, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(70, height, tcell.ButtonNone, 0))
	a.render()
	if !a.output.Follow || a.textSelection.Active() || a.textSelection.Dragging() {
		t.Fatal("thumb drag failed to restore following or captured text")
	}
}

func TestTranscriptScrollSteeringPreservesLaterManualScroll(t *testing.T) {
	a := fixture(t)
	a.historyEnd = true
	a.output.Append(strings.Repeat("earlier history\n", 60), false)
	height := a.chatHeight(80, a.bodyHeight(24))
	a.scroll(-3, 80, height)
	a.worker = workerAgent
	a.steering = make(chan string, 1)
	a.editor.Set("Use the existing API")
	a.steerInput(a.editor.String())
	if !a.output.Follow {
		t.Fatal("sending a direction change did not return to the bottom")
	}
	a.scroll(-3, 80, height)
	a.receiveRuntime(&run.Event{Kind: "user_message", Text: "Use the existing API"})
	if a.output.Follow {
		t.Fatal("delayed acknowledgement overrode later manual scrolling")
	}
}

func TestTranscriptScrollbarReflowsAndKeepsCopyLiteral(t *testing.T) {
	a := fixture(t)
	a.historyEnd = true
	text := strings.Repeat("한글 text ", 10)
	a.output.Append(strings.Repeat(text+"\n", 30), false)
	a.render()
	assertForegroundOnly(t, a.screen)
	a.mouse(tcell.NewEventMouse(0, 2, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(79, 3, tcell.Button1, 0))
	a.mouse(tcell.NewEventMouse(79, 3, tcell.ButtonNone, 0))
	if copied := a.textSelection.Text(); copied == "" || strings.ContainsAny(copied, "│┃") {
		t.Fatalf("scrollbar contaminated copied text: %q", copied)
	}
	a.clearSelection()
	a.screen.SetSize(24, 12)
	a.render()
	height := a.chatHeight(24, a.bodyHeight(12))
	if a.scrollbar.x != 23 || a.scrollbar.bar.Height != height || !a.output.Follow {
		t.Fatal("resize did not reflow the scrollbar and preserve following")
	}
	for y := 1; y <= height; y++ {
		if text, _, width := a.screen.Get(23, y); width != 1 || text != "│" && text != "┃" {
			t.Fatalf("scrollbar overwritten at row %d: %q width=%d", y, text, width)
		}
	}
}
