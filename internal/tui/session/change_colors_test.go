// Purpose: Verify applied file diffs keep their colors through restoration, wrapping and folding.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run TestAppliedDiff.
// Related: internal/tui/transcript/events.go, internal/tui/viewport/viewport.go, internal/tui/view/chat.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func TestAppliedDiffColors(t *testing.T) {
	for _, name := range []string{"edit_file", "write_file"} {
		for _, restored := range []bool{false, true} {
			a := screenFixture(t)
			a.screen.SetSize(44, 60)
			event := runrecord.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: name, Arguments: []byte(`{"path":"a"}`)}, Output: []byte(`{"changed":true}`), ChangePreview: "--- a\n+++ a\n@@ -1,2 +1,2 @@\n context\n-" + strings.Repeat("old ", 24) + "\n+" + strings.Repeat("new ", 24) + "\n"}
			if restored {
				a.output.Prepend((transcript.Formatter{Language: a.language, Redact: a.opts.Vault.Redact}).Event(event))
			} else {
				a.receiveRuntime(&event)
			}
			a.render()
			assertDiffRows(t, a)
			a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
			a.render()
			assertForegroundOnly(t, a.screen)
			a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlO, "", 0))
			a.screen.SetSize(80, 60)
			a.render()
			assertDiffRows(t, a)
		}
	}
}

func assertDiffRows(t *testing.T, a *app) {
	t.Helper()
	w, h := a.screen.Size()
	removed, added := 0, 0
	var change byte
	for y := 1; y < h-a.footerRows-1; y++ {
		text := screenLine(a.screen, y, w)
		trimmed := strings.TrimSpace(text)
		switch {
		case strings.HasPrefix(trimmed, "-   2 |"):
			change = '-'
		case strings.HasPrefix(trimmed, "+   2 |"):
			change = '+'
		case trimmed == "":
			change = 0
		}
		fg, bg := tcell.ColorDefault, tcell.ColorDefault
		switch change {
		case '-':
			fg, bg = tcell.NewHexColor(0xe6edf3), tcell.NewHexColor(0x6b2230)
			removed++
		case '+':
			fg, bg = tcell.NewHexColor(0xe6edf3), tcell.NewHexColor(0x14532d)
			added++
		}
		for _, x := range []int{4, view.TranscriptWidth(w) - 1} {
			_, style, _ := a.screen.Get(x, y)
			if style.GetForeground() != fg || style.GetBackground() != bg {
				t.Fatalf("%q at %d,%d: fg=%v bg=%v; want fg=%v bg=%v", text, x, y, style.GetForeground(), style.GetBackground(), fg, bg)
			}
		}
		_, gutter, _ := a.screen.Get(view.TranscriptWidth(w), y)
		if gutter.GetBackground() != tcell.ColorDefault {
			t.Fatal("diff background leaked into scrollbar gutter")
		}
		_, margin, _ := a.screen.Get(0, y)
		if margin.GetBackground() != tcell.ColorDefault {
			t.Fatal("diff background covered tool indentation")
		}
	}
	if removed < 2 || added < 2 {
		t.Fatalf("wrapped changed rows missing: removed=%d added=%d", removed, added)
	}
}

func TestAppliedDiffDoesNotColorPlainToolOutput(t *testing.T) {
	a := screenFixture(t)
	a.receiveRuntime(&runrecord.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "execute_command"}, Output: []byte(`{"Stdout":"-   1 | old\n+   1 | new\n","ExitCode":0}`)})
	a.render()
	assertForegroundOnly(t, a.screen)
	a.receiveRuntime(&runrecord.Event{Kind: "tool_failed", Text: "failed", Call: &tool.Call{Name: "edit_file"}, Output: []byte(`{"changed":true}`), ChangePreview: "--- a\n+++ a\n@@ -1 +1 @@\n-old\n+new\n"})
	a.render()
	assertForegroundOnly(t, a.screen)
}
