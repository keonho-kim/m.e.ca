// Purpose: Verify partial header refreshes match full frames and preserve body, input and selection.
// Pattern: Virtual terminal rendering tests.
// Usage: go test ./internal/tui/session -run TestMemoryHeader.
// Related: internal/tui/session/render_header.go, internal/tui/view/tokens.go, internal/tui/session/text_selection.go.
package session

import (
	"slices"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestMemoryHeaderMatchesFullScreenAcrossViews(t *testing.T) {
	a := screenFixture(t)
	a.proposal = &toolapi.Approval{Name: "execute_command", Target: "project"}
	for _, page := range []string{"chat", "model", "language", "permission", "files", "approval", "quit"} {
		a.page = page
		a.render()
		for _, value := range []uint64{1 << 40, 42 << 20, 0} {
			a.memory.read = func() (uint64, error) { return value, nil }
			a.memory.sample()
			body := screenRows(a, 24)[1:]
			a.renderHeader()
			if !slices.Equal(body, screenRows(a, 24)[1:]) {
				t.Fatalf("%s memory refresh changed the body", page)
			}
			partial := screenLine(a.screen, 0, 80)
			a.render()
			if full := screenLine(a.screen, 0, 80); partial != full {
				t.Fatalf("%s header mismatch:\npartial=%q\nfull=%q", page, partial, full)
			}
		}
	}
}

func TestMemoryHeaderPreservesInputSelectionAndScroll(t *testing.T) {
	a := screenFixture(t)
	a.compose.editor.Set("draft text")
	a.chat.output.Append("kept transcript\n", false)
	a.chat.output.Follow = false
	a.render()
	r := a.inputRegion
	drag(a, 2, r.y, 6, r.y)
	a.render()
	selected := a.textSelection.Text()
	_, style, _ := a.screen.Get(2, r.y)
	visible := &a.chat.chatVisible[0]
	a.memory.read = func() (uint64, error) { return 42 << 20, nil }
	a.memory.sample()
	a.renderHeader()
	_, refreshedStyle, _ := a.screen.Get(2, r.y)
	if !a.textSelection.Active() || selected != a.textSelection.Text() || style != refreshedStyle ||
		a.compose.editor.String() != "draft text" || a.inputRegion.editor != r.editor ||
		a.chat.output.Follow || visible != &a.chat.chatVisible[0] {
		t.Fatal("memory refresh changed selection, input or transcript projection")
	}
	// The preserved source mapping must still support an actual subsequent edit.
	if !a.deleteSelection() || a.compose.editor.String() != " text" {
		t.Fatal("memory refresh invalidated selected input coordinates")
	}
}

func TestMemoryHeaderPreservesSmallTerminalPrompt(t *testing.T) {
	a := screenFixture(t)
	for _, size := range [][2]int{{8, 3}, {80, 4}, {80, 5}} {
		a.screen.SetSize(size[0], size[1])
		for _, page := range []string{"model", "language", "permission"} {
			a.page = page
			a.render()
			want := screenLine(a.screen, 0, size[0])
			a.memory.read = func() (uint64, error) { return 42 << 20, nil }
			a.memory.sample()
			a.renderHeader()
			if got := screenLine(a.screen, 0, size[0]); got != want {
				t.Fatalf("%s %v: memory refresh overwrote the resize prompt: %q -> %q", page, size, want, got)
			}
		}
	}
}
