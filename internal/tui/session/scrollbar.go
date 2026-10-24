// Purpose: Route transcript scrollbar clicks and captured thumb drags before text selection.
// Pattern: Event-loop interaction controller.
// Usage: mouse dispatches scrollbarMouse; renderChat supplies the visible geometry.
// Related: internal/tui/session/keys.go, internal/tui/session/render_chat.go, internal/tui/view/chat.go, internal/tui/viewport/scrollbar.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/tui/view"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type scrollbarState struct {
	bar        viewport.Scrollbar
	x, y, grab int
	dragging   bool
}

func (a *app) scrollbarMouse(e *tcell.EventMouse) bool {
	if a.page != "chat" {
		return false
	}
	state := &a.scrollbar
	if e.Buttons()&(tcell.WheelUp|tcell.WheelDown) != 0 {
		state.dragging = false
		return false
	}
	if !state.dragging && a.textSelection.Dragging() {
		return false
	}
	x, y := e.Position()
	if state.dragging {
		a.clearSelection()
		w, _ := a.screen.Size()
		transcript := a.chatLayout(view.TranscriptWidth(w))
		bar := viewport.NewScrollbar(transcript.Count(), state.bar.Height, 0)
		transcript.Seek(&a.output, bar.Index(y-state.y, state.grab), state.bar.Height)
		state.dragging = e.Buttons()&tcell.Button1 != 0
		if !a.output.Follow && transcript.Start(&a.output, state.bar.Height) == 0 {
			a.loadOlderHistory()
		}
		return true
	}
	if e.Buttons()&tcell.Button1 == 0 || x != state.x || y < state.y || y >= state.y+state.bar.Height {
		return false
	}
	a.clearSelection()
	if !state.bar.Visible() {
		return true
	}
	row := y - state.y
	if row >= state.bar.ThumbTop && row < state.bar.ThumbTop+state.bar.ThumbRows {
		state.dragging, state.grab = true, row-state.bar.ThumbTop
	} else {
		distance := max(1, state.bar.Height-1)
		if row < state.bar.ThumbTop {
			distance = -distance
		}
		w, _ := a.screen.Size()
		a.scroll(distance, w, state.bar.Height)
	}
	return true
}
