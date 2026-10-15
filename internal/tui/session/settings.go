// Purpose: Apply current session or global policy and Thinking settings with explicit persistence.
// Pattern: Use Case.
// Usage: Called after the settings destination menu chooses session or global ownership.
// Related: internal/config/config.go, internal/storage/sqlite/selection.go, internal/tui/session/menus.go.
package session

import (
	settingcmd "github.com/keonho-kim/v-code/internal/commands/settings"
)

func (a *app) applySetting(global bool) {
	if a.busy {
		a.status = "Stop the current task before changing settings"
		return
	}
	if a.changeField == "thinking" {
		var err error
		a.opts.Config, a.selection, err = settingcmd.Thinking(a.ctx, a.opts.Store, a.opts.SaveConfig, a.opts.Session.ID, a.opts.Config, a.selection, a.changeValue == "ON", global)
		a.fail(err)
		return
	}
	updated, effective, err := settingcmd.Policy(a.ctx, a.opts.Store, a.opts.SaveConfig, a.opts.Config, a.changeValue, global)
	if err != nil {
		a.fail(err)
		return
	}
	a.opts.Config, a.settings = updated, effective
}
