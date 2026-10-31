// Purpose: Process input immediately and coalesce terminal repaint work into active frames.
// Pattern: Input-prioritized event loop with an adaptive one-shot render timer.
// Usage: Run invokes loop after restoration; shutdown joins workers and releases timers.
// Related: internal/tui/session/app.go, internal/tui/session/frame_pacer.go, internal/tui/session/paste.go, internal/tui/session/render.go.
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
	started := time.Now()
	a.render()
	var pacer framePacer
	pacer.painted(started, time.Now(), false)
	inputPending := false
	preferInput := true
	for !a.exit && a.ctx.Err() == nil {
		var terminal tcell.Event
		// Give ready input the first turn, then fairly service runtime notices
		// before allowing another priority batch. Continuous input cannot starve them.
		if preferInput {
			select {
			case terminal = <-input:
			default:
			}
		}
		if terminal == nil {
			select {
			case <-a.ctx.Done():
				a.exit = true
			case terminal = <-input:
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
		}
		if terminal != nil {
			preferInput = false
			inputPending = true
			if err := a.terminalBatch(terminal, input); err != nil {
				return err
			}
		} else {
			preferInput = true
		}
		a.syncFileWatch()
		if a.exit || a.paste {
			continue
		}
		remaining := pacer.delay(time.Now(), inputPending)
		if remaining <= 0 {
			timer.Stop()
			frame = nil
			started := time.Now()
			a.render()
			pacer.painted(started, time.Now(), inputPending)
			inputPending = false
		} else if frame == nil || inputPending {
			timer.Reset(remaining)
			frame = timer.C
		}
	}
	return nil
}

func (a *app) terminalBatch(first tcell.Event, input <-chan tcell.Event) error {
	event := first
	for count := 0; count < 32; count++ {
		if err := a.terminalEvent(event); err != nil {
			return err
		}
		if count == 31 || a.exit || a.ctx.Err() != nil {
			return nil
		}
		select {
		case event = <-input:
		default:
			return nil
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
		if a.queued.Edit != nil {
			a.queued.Edit.Reveal = true
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
