// Purpose: Apply current project or global policy and Thinking settings with explicit persistence.
// Pattern: Use Case.
// Usage: Called after the settings destination menu chooses project/session or global ownership.
// Related: internal/config/config.go, internal/storage/sqlite/selection.go, internal/tui/session/menus.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (a *app) applySetting(global bool) {
	if a.busy {
		a.status = "Stop the current task before changing settings"
		return
	}
	if a.changeField == "thinking" {
		value := a.changeValue == "ON"
		a.selection.Thinking = &value
		if global {
			c := a.opts.Config
			c.Thinking = value
			if err := config.Save(a.opts.State, c); err != nil {
				a.fail(err)
				return
			}
			a.opts.Config = c
		}
		a.fail(a.opts.Store.SaveSelection(a.ctx, a.opts.Session.ID, a.selection))
		return
	}
	project, err := a.opts.Store.ProjectSettings(a.ctx, a.opts.Session.ProjectRoot)
	if err != nil {
		a.fail(err)
		return
	}
	c := a.opts.Config
	if a.changeField == "permission" {
		v, err := policy.ParsePermission(a.changeValue)
		if err != nil {
			a.fail(err)
			return
		}
		if global {
			c.Defaults.Permission = v
		} else {
			project.Permission = v
		}
	} else {
		v, err := policy.ParseScope(a.changeValue)
		if err != nil {
			a.fail(err)
			return
		}
		if global {
			c.Defaults.Scope = v
		} else {
			project.Scope = v
		}
	}
	if global {
		err = config.Save(a.opts.State, c)
	} else {
		err = a.opts.Store.SaveProjectSettings(a.ctx, a.opts.Session.ProjectRoot, project)
	}
	if err != nil {
		a.fail(err)
		return
	}
	a.opts.Config = c
	a.settings, err = config.Effective(c.Defaults, project)
	a.fail(err)
	a.closeReferences()
}
