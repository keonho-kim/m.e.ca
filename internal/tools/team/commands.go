// Purpose: Forward command control while releasing shared shell access only after owned processes settle.
// Pattern: Command-lifecycle decorator.
// Usage: The runtime calls command interfaces on Executor throughout one peer lifetime.
// Related: internal/tools/team/local.go, internal/tools/team/workspace/workspace.go.
package team

import (
	"context"
	"encoding/json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
	"time"
)

func (e *Executor) BeginCommands(ctx context.Context, deadline time.Time) {
	e.Local.BeginCommands(ctx, deadline)
}
func (e *Executor) StopCommands() {
	e.Local.StopCommands()
	e.commands.Wait()
	e.releaseShell()
}
func (e *Executor) CommandUpdates() <-chan struct{}       { return e.Local.CommandUpdates() }
func (e *Executor) HasCommands() bool                     { return e.Local.HasCommands() }
func (e *Executor) CommandActivity() []toolapi.CommandRun { return e.Local.CommandActivity() }
func (e *Executor) CollectCommands() ([]toolapi.CommandCompletion, error) {
	out, err := e.Local.CollectCommands()
	e.releaseFinishedShell()
	return out, err
}
func (e *Executor) CommandCapture(id string) (json.RawMessage, error) {
	return e.Local.CommandCapture(id)
}
func (e *Executor) AcknowledgeCommand(id string) error { return e.Local.AcknowledgeCommand(id) }

// holdShell transfers exactly this lease until the command settles. It never
// consumes the runtime's completion notification or writes its journal.
func (e *Executor) holdShell(id string, held *workspace.Lease) error {
	e.mu.Lock()
	e.shell = held
	e.mu.Unlock()
	done, err := e.Local.CommandDone(id)
	if err != nil {
		return err
	}
	e.commands.Go(func() {
		<-done
		e.mu.Lock()
		if e.shell == held {
			e.shell = nil
		}
		e.mu.Unlock()
		held.Release()
	})
	return nil
}
func (e *Executor) releaseFinishedShell() {
	if e.Local.ActiveCommand() != "" {
		return
	}
	e.releaseShell()
}
func (e *Executor) releaseShell() {
	e.mu.Lock()
	held := e.shell
	e.shell = nil
	e.mu.Unlock()
	held.Release()
}
