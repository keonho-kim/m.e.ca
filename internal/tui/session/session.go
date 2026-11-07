// Purpose: Restore the model, conversation, selected paths and configured execution policy.
// Pattern: Session recovery use case.
// Usage: Called before the terminal event loop starts.
// Related: internal/agent/session/recovery/recovery.go, internal/tui/session/app.go.
package session

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/policy"
)

func (a *app) restore() error {
	var err error
	ctx := a.ctx
	if a.opts.LoadTeamMode != nil {
		a.team.enabled, err = a.opts.LoadTeamMode(ctx)
		if err != nil {
			return err
		}
	}
	a.chat.tokenTotal, err = a.opts.Store.TokenUsage(ctx, a.opts.Session.ID)
	if err != nil {
		return err
	}
	a.settings = policy.Settings{Scope: policy.System, Permission: a.opts.Config.Permission}
	a.compose.inputs, err = a.opts.Store.RecentInputs(ctx, a.opts.Session.ID, 200)
	if err != nil {
		return err
	}
	if err := a.loadHistory(); err != nil {
		return err
	}
	cp, taskErr := a.opts.Store.LatestRun(ctx, a.opts.Session.ID)
	if taskErr != nil {
		return taskErr
	}
	if cp != nil && cp.Status == runrecord.Interrupted {
		a.chat.output.Append(a.language.Format("Interrupted work: %s\nSend a message to inspect the current state and continue.", a.opts.Vault.Redact(cp.Objective)), false)
	}
	a.compose.historyIndex = len(a.compose.inputs)
	a.connection.selection, err = a.opts.Store.Selection(ctx, a.opts.Session.ID)
	if err != nil {
		return err
	}
	if a.connection.selection.Model == "" && a.opts.Config.Model != nil {
		a.connection.selection, err = a.opts.Config.Model.Selection()
		if err != nil {
			return err
		}
	}
	if a.connection.selection.Model != "" {
		// The setup worker owns credential I/O so an OS unlock prompt never
		// prevents the terminal event loop from processing cancellation.
		a.connect(a.connection.selection.Connection, "", a.connection.selection.Model, false)
	}
	paths, err := a.opts.Store.FileSelections(ctx, a.opts.Session.ID)
	if err != nil {
		return err
	}
	a.files.browser.Selected = map[string]bool{}
	for _, path := range paths {
		if _, err := a.opts.Files.Reference(a.ctx, a.settings, path); err == nil {
			a.files.browser.Selected[path] = true
		}
	}
	return nil
}
