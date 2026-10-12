// Purpose: Own terminal session state and join all input and task workers on shutdown.
// Pattern: Event Loop.
// Usage: cmd/v supplies initialized adapters and calls Run on a tcell screen.
// Related: internal/tui/session/keys.go, internal/tui/session/tasks.go, internal/tui/session/render.go, internal/tui/session/events.go.
package session

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/keonho-kim/v-code/internal/agent"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/commands"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/config/credentials"
	"github.com/keonho-kim/v-code/internal/policy"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/keonho-kim/v-code/internal/tools/local"
	"github.com/keonho-kim/v-code/internal/tui/browser"
	"github.com/keonho-kim/v-code/internal/tui/input"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
)

type Options struct {
	Home, State string
	Config      config.Config
	Store       *sqlite.Store
	Session     sqlite.Session
	Tools       *local.Executor
	Vault       *credentials.Vault
}
type app struct {
	opts                            Options
	screen                          tcell.Screen
	ctx                             context.Context
	cancel                          context.CancelFunc
	workers                         sync.WaitGroup
	events                          chan notice
	editor                          input.Editor
	output                          viewport.View
	history                         []agent.Message
	inputs                          []string
	historyIndex                    int
	draft                           string
	settings                        policy.Settings
	selection                       sqlite.Selection
	client                          *httpclient.Client
	plan                            *planning.Plan
	planMode                        bool
	page                            string
	choices                         []string
	choice                          int
	menuInput                       input.Editor
	connectionKind                  int
	endpoint, key                   string
	browser                         browser.Browser
	proposal                        *agent.ApprovalRequest
	decision                        chan bool
	busy                            bool
	taskCancel                      context.CancelFunc
	exit                            bool
	paste                           bool
	status, stream, thinking        string
	refs                            []local.Reference
	refOpen                         bool
	refStart, refIndex, refRevision int
	refCancel                       context.CancelFunc
	commandHints                    []commands.Suggestion
	commandOpen                     bool
	commandIndex                    int
	changeField, changeValue        string
	historyCursor                   int64
	previewOffset                   int
	pendingConnection               modelconn.Connection
}

func Run(ctx context.Context, screen tcell.Screen, opts Options) (err error) {
	app := &app{opts: opts, screen: screen, events: make(chan notice, 64), page: "chat", output: viewport.View{Follow: true}}
	app.ctx, app.cancel = context.WithCancel(ctx)
	defer func() { app.cancel(); app.workers.Wait() }()
	if err = app.restore(); err != nil {
		return err
	}
	inputEvents := make(chan tcell.Event, 32)
	quit := make(chan struct{})
	joined := make(chan struct{})
	go func() { defer close(joined); screen.ChannelEvents(inputEvents, quit) }()
	defer func() { close(quit); <-joined }()
	screen.EnableMouse(tcell.MouseButtonEvents)
	screen.EnablePaste()
	for !app.exit {
		app.render()
		select {
		case <-app.ctx.Done():
			return nil
		case event := <-inputEvents:
			switch e := event.(type) {
			case *tcell.EventKey:
				app.keyEvent(e)
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventMouse:
				app.mouse(e)
			case *tcell.EventPaste:
				app.paste = e.Start()
				app.refreshHints()
			case *tcell.EventError:
				return e
			}
		case n := <-app.events:
			app.receive(n)
		}
	}
	return nil
}
func (a *app) mouse(e *tcell.EventMouse) {
	n := 0
	if e.Buttons()&tcell.WheelUp != 0 {
		n = -3
	}
	if e.Buttons()&tcell.WheelDown != 0 {
		n = 3
	}
	if n == 0 {
		return
	}
	if a.page == "files" {
		a.browser.Move(n)
		return
	}
	w, h := a.screen.Size()
	a.scroll(n, w, h-3)
}
func (a *app) planText() string {
	if a.plan == nil {
		return ""
	}
	text := fmt.Sprintf("Plan revision %d: %s\n", a.plan.Revision, a.plan.Objective)
	for i, step := range a.plan.Steps {
		text += fmt.Sprintf("%d. %s [%s]\n", i+1, step.Description, step.Status)
	}
	return text + string(a.plan.Status)
}
func (a *app) fail(err error) {
	if err != nil && !errors.Is(err, context.Canceled) {
		a.status = a.opts.Vault.Redact(err.Error())
	}
}
