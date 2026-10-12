// Purpose: Restore session policy, model, Plan and selected paths without replaying effects.
// Pattern: Session recovery use case.
// Usage: Called before the terminal event loop starts.
// Related: internal/storage/sqlite/recovery.go, internal/tui/session/app.go.
package session

import (
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/config"
)

func (a *app) restore() error {
	var err error
	ctx := a.ctx
	project, err := a.opts.Store.ProjectSettings(ctx, a.opts.Session.ProjectRoot)
	if err != nil {
		return err
	}
	a.settings, err = config.Effective(a.opts.Config.Defaults, project)
	if err != nil {
		return err
	}
	a.history, err = a.opts.Store.Messages(ctx, a.opts.Session.ID, 1000)
	if err != nil {
		return err
	}
	for _, m := range a.history {
		if m.Role == "user" {
			a.inputs = append(a.inputs, m.Content)
		}
	}
	if len(a.inputs) > 200 {
		a.inputs = a.inputs[len(a.inputs)-200:]
	}
	if err := a.loadHistory(false); err != nil {
		return err
	}
	cp, taskErr := a.opts.Store.LatestTask(ctx, a.opts.Session.ID)
	if taskErr != nil {
		return taskErr
	}
	if cp != nil && cp.Status == run.Interrupted {
		a.output.Append("Interrupted task: "+a.opts.Vault.Redact(cp.Objective)+"\nPrevious phase: "+cp.Phase+". Send a new request to continue after inspecting current state.", false)
	}
	a.historyIndex = len(a.inputs)
	a.plan, err = a.opts.Store.LatestPlan(ctx, a.opts.Session.ID)
	if err != nil {
		return err
	}
	if a.plan != nil {
		a.planMode = a.plan.Status == planning.Planning || a.plan.Status == planning.AwaitingApproval
		if a.planMode {
			a.output.Append(a.planText(), false)
		}
	}
	a.selection, err = a.opts.Store.Selection(ctx, a.opts.Session.ID)
	if err != nil {
		return err
	}
	if a.selection.Model != "" {
		key, keyErr := a.opts.Vault.Get(a.selection.Connection.CredentialRef)
		if keyErr != nil {
			a.status = keyErr.Error()
		} else {
			// Connection validation runs before a generation client is installed.
			a.connect(a.selection.Connection, key, a.selection.Model)
		}
	}
	paths, err := a.opts.Store.FileSelections(ctx, a.opts.Session.ID)
	if err != nil {
		return err
	}
	a.browser.Selected = map[string]bool{}
	for _, path := range paths {
		if _, err := a.opts.Tools.Reference(a.settings, path); err == nil {
			a.browser.Selected[path] = true
		}
	}
	return nil
}
