// Purpose: Restore the model, conversation and selected paths with fixed application access policy.
// Pattern: Session recovery use case.
// Usage: Called before the terminal event loop starts.
// Related: internal/storage/recovery/recovery.go, internal/tui/session/app.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (a *app) restore() error {
	var err error
	ctx := a.ctx
	a.tokenTotal, err = a.opts.Store.TokenUsage(ctx, a.opts.Session.ID)
	if err != nil {
		return err
	}
	a.settings = policy.Settings{Scope: policy.System}
	recent, err := a.opts.Store.Messages(ctx, a.opts.Session.ID, 1000)
	if err != nil {
		return err
	}
	for _, m := range recent {
		if m.Role == "user" {
			a.inputs = append(a.inputs, m.Content)
		}
	}
	if len(a.inputs) > 200 {
		clear(a.inputs[:len(a.inputs)-200])
		a.inputs = a.inputs[len(a.inputs)-200:]
	}
	if err := a.loadHistory(false); err != nil {
		return err
	}
	cp, taskErr := a.opts.Store.LatestRun(ctx, a.opts.Session.ID)
	if taskErr != nil {
		return taskErr
	}
	if cp != nil && cp.Status == run.Interrupted {
		a.output.Append(a.language.Format("Interrupted work: %s\nSend a message to inspect the current state and continue.", a.opts.Vault.Redact(cp.Objective)), false)
	}
	a.historyIndex = len(a.inputs)
	a.selection, err = a.opts.Store.Selection(ctx, a.opts.Session.ID)
	if err != nil {
		return err
	}
	if a.selection.Model == "" && a.opts.Config.Model != nil {
		a.selection, err = a.opts.Config.Model.Selection()
		if err != nil {
			return err
		}
	}
	if a.selection.Model != "" {
		key, keyErr := a.opts.Vault.Get(a.selection.Connection.CredentialRef)
		if keyErr != nil {
			a.status = keyErr.Error()
		} else {
			// Resolve current server metadata and configured controls before adopting the client.
			a.connect(a.selection.Connection, key, a.selection.Model, false)
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
