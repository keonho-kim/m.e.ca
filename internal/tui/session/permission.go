// Purpose: Stage, confirm and save the Default or All Mighty approval mode.
// Pattern: Event-loop settings use case.
// Usage: !permission opens the shared settings screen; changes apply to future requests.
// Related: internal/commands/settings/permission.go, internal/tui/view/settings.go, internal/tui/session/confirmation.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	settingcmd "github.com/keonho-kim/meca/internal/commands/settings"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func (a *app) openPermission() {
	a.cancelDirectory()
	a.stopNotice, a.status = "", ""
	a.page, a.choice = "permission", int(a.opts.Config.Permission)
	a.permissionConfirm = false
	a.menuInput.Set("")
}

func (a *app) permissionKey(e *tcell.EventKey) {
	if a.permissionConfirm {
		switch confirmationKey(e) {
		case confirmEdit:
			a.permissionConfirm = false
		case confirmAccept:
			a.permissionConfirm = false
			a.savePermission()
		case confirmReject:
			a.permissionConfirm = false
			a.page, a.status = "chat", ""
		}
		return
	}
	switch e.Key() {
	case tcell.KeyRight, tcell.KeyLeft:
		a.choice = 1 - a.choice
	case tcell.KeyEnter:
		a.permissionConfirm = true
	case tcell.KeyEscape:
		a.page, a.status = "chat", ""
	}
}

func (a *app) savePermission() {
	if a.worker.active() {
		a.status = a.language.Text("Stop the current task before changing settings")
		return
	}
	next, err := settingcmd.Permission(a.opts.Config, policy.Permission(a.choice), a.opts.SaveConfig)
	if err != nil {
		a.fail(err)
		return
	}
	a.opts.Config = next
	a.settings.Permission = next.Permission
	a.page, a.status = "chat", ""
}

func (a *app) permissionShortcuts() []view.Shortcut {
	if a.permissionConfirm {
		return settingsConfirmationShortcuts()
	}
	return []view.Shortcut{{Key: "^C", Label: "Exit"}, {Key: "Esc", Label: "Close"}, {Key: "←→", Label: "Select"}, {Key: "Enter", Label: "Review save"}}
}

func (a *app) renderPermission(w, h int) {
	description := "Run ordinary work automatically; ask for approval when required."
	if a.choice == int(policy.PermissionAllMighty) {
		description = "Automatically approve operations without asking."
	}
	view.Settings(a.screen, view.SettingsDisplay{
		Title: a.language.Text("Permission mode"), Language: a.language,
		Activity: a.activity(), Usage: a.displayTokens(), Status: a.status,
		Confirm: a.permissionConfirm,
		Rows:    []view.SettingRow{{Label: a.language.Text("Permission mode"), Values: []string{a.language.Text("Default"), "All Mighty"}, Selected: a.choice}},
		Details: []string{a.language.Text(description)},
	}, w, h)
}
