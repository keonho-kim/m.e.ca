// Purpose: Process input immediately and coalesce terminal repaint work into active frames.
// Pattern: Event loop with an owned one-shot render timer.
// Usage: Run invokes loop after restoration; shutdown joins workers and releases timers.
// Related: internal/tui/session/app.go, internal/tui/session/paste.go, internal/tui/session/render.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"time"
)

const frameInterval = time.Second / 60

func (a *app) loop() error {
	input := a.screen.EventQ()
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()
	var frame <-chan time.Time
	a.syncFileWatch()
	a.render()
	last := time.Now()
	for !a.exit {
		select {
		case <-a.ctx.Done():
			a.exit = true
		case event := <-input:
			if err := a.terminalEvent(event); err != nil {
				return err
			}
		case <-a.elapsedTicks:
			a.updateElapsed(time.Now())
		case <-a.compressionTicks:
			a.advanceCompression()
		case n := <-a.events:
			a.receive(n)
		case event, ok := <-a.fileWatch.events:
			if ok {
				a.observeFileChange(event)
			} else {
				a.fileWatchFailed(nil)
			}
		case err, ok := <-a.fileWatch.errors:
			if ok {
				a.fileWatchFailed(err)
			} else {
				a.fileWatchFailed(nil)
			}
		case <-a.fileWatch.ticks:
			a.refreshWatchedFiles()
		case <-frame:
			frame = nil
		}
		a.syncFileWatch()
		if a.exit || a.paste {
			continue
		}
		remaining := frameInterval - time.Since(last)
		if remaining <= 0 {
			timer.Stop()
			frame = nil
			a.render()
			last = time.Now()
		} else if frame == nil {
			timer.Reset(remaining)
			frame = timer.C
		}
	}
	return nil
}

func (a *app) terminalEvent(event tcell.Event) error {
	switch e := event.(type) {
	case *tcell.EventKey:
		a.keyEvent(e)
	case *tcell.EventResize:
		a.clearSelection()
		if a.pendingEdit != nil {
			a.pendingEdit.reveal = true
		}
		a.screen.Sync()
	case *tcell.EventMouse:
		a.mouse(e)
	case *tcell.EventPaste:
		a.pasteBoundary(e.Start())
	case *tcell.EventError:
		return e
	}
	return nil
}
