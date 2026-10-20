// Purpose: Apply current session or global Thinking settings with explicit persistence.
// Pattern: Use Case.
// Usage: Called after the settings destination menu chooses session or global ownership.
// Related: internal/config/config.go, internal/storage/sqlite/selection.go, internal/tui/session/menus.go.
package session

import (
	settingcmd "github.com/keonho-kim/v-code/internal/commands/settings"
)

func (a *app) applySetting(global bool) {
	if a.busy {
		a.status = a.language.Text("Stop the current task before changing settings")
		return
	}
	var err error
	a.opts.Config, a.selection, err = settingcmd.Thinking(a.ctx, a.opts.Store, a.opts.SaveConfig, a.opts.Session.ID, a.opts.Config, a.selection, a.thinkingChoice, global)
	a.fail(err)
}
