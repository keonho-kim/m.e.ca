// Purpose: Verify a real terminal event queue batches paste without losing text or painting each character.
// Pattern: Virtual terminal event-loop test.
// Usage: go test ./internal/tui/session -run TestEventLoopPaste.
// Related: internal/tui/session/event_loop.go, internal/tui/session/paste.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"strings"
	"testing"
	"time"
)

type frameScreen struct {
	tcell.Screen
	frames int
}

func (s *frameScreen) Show() { s.frames++; s.Screen.Show() }

func TestEventLoopPasteBatchesFrames(t *testing.T) {
	a := fixture(t)
	screen := &frameScreen{Screen: a.screen}
	a.screen = screen
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	a.ctx = ctx
	text := strings.Repeat("한글 a\u0301 👩‍💻 ", 100)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		send := func(event tcell.Event) bool {
			select {
			case screen.EventQ() <- event:
				return true
			case <-ctx.Done():
				return false
			}
		}
		if !send(tcell.NewEventPaste(true)) {
			return
		}
		for _, r := range text {
			if !send(tcell.NewEventKey(tcell.KeyRune, string(r), 0)) {
				return
			}
		}
		if !send(tcell.NewEventPaste(false)) {
			return
		}
		send(tcell.NewEventKey(tcell.KeyCtrlC, "", 0))
	}()
	if err := a.loop(); err != nil {
		t.Fatal(err)
	}
	<-finished
	if ctx.Err() == context.DeadlineExceeded || a.editor.String() != text {
		t.Fatalf("paste was lost or input queue stalled: error=%v bytes=%d want=%d", ctx.Err(), len(a.editor.String()), len(text))
	}
	if screen.frames > 4 {
		t.Fatalf("paste repainted %d frames", screen.frames)
	}
}
