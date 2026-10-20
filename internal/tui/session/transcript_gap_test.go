// Purpose: Keep five empty cells between wrapped transcript content and the working scrollbar.
// Pattern: Virtual terminal regression tests.
// Usage: go test ./internal/tui/session -run TestTranscriptScrollbarGap.
// Related: internal/tui/session/render_chat.go, internal/tui/session/scrollbar.go, internal/tui/view/chat.go.
package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
)

func TestTranscriptScrollbarGap(t *testing.T) {
	for _, width := range []int{80, 24} {
		for _, markdown := range []bool{false, true} {
			t.Run(fmt.Sprintf("width=%d/markdown=%t", width, markdown), func(t *testing.T) {
				a := fixture(t)
				a.historyEnd = true
				a.screen.SetSize(width, 24)
				text := strings.Repeat("x", 4096)
				if markdown {
					a.stream = "**" + text + "**"
				} else {
					a.thinking = text
				}
				a.render()
				content := width - 6
				if !a.scrollbar.bar.Visible() {
					t.Fatal("scrollbar missing")
				}
				for y := 1; y <= a.scrollbar.bar.Height; y++ {
					for x := content; x < width-1; x++ {
						glyph, _, _ := a.screen.Get(x, y)
						if strings.TrimSpace(glyph) != "" {
							t.Fatalf("reserved gap occupied at (%d,%d): %q", x, y, glyph)
						}
					}
				}
				if first := screenLine(a.screen, 1, content); first != strings.Repeat("x", content) {
					t.Fatalf("content clipped or has extra padding: %q", first)
				}
				if a.inputRegion.width != width {
					t.Fatal("transcript margin narrowed the input")
				}
				before := a.chatVisible[0].Row
				a.mouse(tcell.NewEventMouse(0, 2, tcell.WheelUp, 0))
				a.render()
				if a.output.Follow || a.chatVisible[0].Row != before-3 {
					t.Fatal("scrolling used a different wrap width")
				}
				drag(a, 0, 1, width-2, 2)
				if copied := a.textSelection.Text(); copied != strings.Repeat("x", content*2) {
					t.Fatalf("margin changed copied source: %q", copied)
				}
				a.clearSelection()
				a.render()
				thumb := a.scrollbar.y + a.scrollbar.bar.ThumbTop
				a.mouse(tcell.NewEventMouse(width-1, thumb, tcell.Button1, 0))
				a.mouse(tcell.NewEventMouse(width-1, a.scrollbar.y+a.scrollbar.bar.Height-1, tcell.Button1, 0))
				a.mouse(tcell.NewEventMouse(width-1, a.scrollbar.y+a.scrollbar.bar.Height-1, tcell.ButtonNone, 0))
				a.render()
				if !a.output.Follow || a.textSelection.Active() {
					t.Fatal("thumb drag did not return to the actual displayed bottom")
				}
			})
		}
	}
}
