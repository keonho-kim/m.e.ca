// Purpose: Run the terminal event loop and join workers and clients during shutdown.
// Pattern: Event Loop.
// Usage: cmd/v supplies initialized adapters and calls Run on a tcell screen.
// Related: internal/tui/session/keys.go, internal/tui/session/tasks.go, internal/tui/session/render.go, internal/tui/session/events.go.
package session

import (
	"context"
	"errors"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tui/locale"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

type Action uint8

const (
	Exit Action = iota
	NewSession
)

func Run(ctx context.Context, screen tcell.Screen, opts Options) (action Action, err error) {
	app := &app{opts: opts, language: locale.System(ctx), screen: screen, events: make(chan notice, 64), page: "chat", output: viewport.View{Follow: true}}
	app.ctx, app.cancel = context.WithCancel(ctx)
	defer app.shutdown()
	if err = app.restore(); err != nil {
		return Exit, err
	}
	screen.EnableMouse(tcell.MouseDragEvents)
	screen.EnablePaste()
	if err = app.loop(); err != nil {
		return Exit, err
	}
	if app.newSession && ctx.Err() == nil {
		return NewSession, nil
	}
	return Exit, nil
}
func (a *app) fail(err error) {
	if err != nil && !errors.Is(err, context.Canceled) {
		a.status = a.opts.Vault.Redact(err.Error())
	}
}

func (a *app) requestExit() {
	a.cancelDirectory()
	a.closeFilePreview()
	a.stopElapsed()
	a.stopCompression()
	a.exit = true
	if a.taskCancel != nil {
		a.taskCancel()
	}
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *app) shutdown() {
	a.requestExit()
	a.workers.Wait()
	// A connection worker may have transferred a client to a notice just before
	// cancellation. The stopped event loop must release that queued ownership.
	for {
		select {
		case notice := <-a.events:
			if notice.Client != nil && notice.Client != a.client {
				notice.Client.Close()
			}
		default:
			if a.client != nil {
				a.client.Close()
			}
			return
		}
	}
}
