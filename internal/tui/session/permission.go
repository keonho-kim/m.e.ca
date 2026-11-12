// Purpose: Stage, confirm and save the Default or All Mighty approval mode.
// Pattern: Event-loop settings presentation with owned background persistence.
// Usage: !permission opens the shared settings screen; changes apply to future requests.
// Related: internal/commands/settings/permission.go, internal/tui/view/settings.go, internal/tui/session/settings_save.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/config"

	settingcmd "github.com/keonho-kim/meca/internal/commands/settings"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func (a *app) openPermission() {
	a.files.cancelDirectory()
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
	current, value, update := a.opts.Config, policy.Permission(a.choice), a.opts.UpdateConfig
	a.startSettingsSave("permission", func(ctx context.Context) (config.Config, error) {
		return settingcmd.Permission(ctx, current, value, update)
	})
}

func (a *app) permissionShortcuts() []view.Shortcut {
	if a.permissionConfirm {
		return settingsConfirmationShortcuts()
	}
	return []view.Shortcut{{Key: "^C", Label: "Exit"}, {Key: "Esc", Label: "Close"}, {Key: "←→", Label: "Select"}, {Key: "Enter", Label: "Review save"}}
}

func (a *app) renderPermission(w, h int) {
	a.headerTitle = a.language.Text("Permission mode")
	description := "Run ordinary work automatically; ask for approval when required."
	if a.choice == int(policy.PermissionAllMighty) {
		description = "Automatically approve operations; outbound transfers still require your approval."
	}
	view.Settings(a.screen, view.SettingsDisplay{
		Title: a.headerTitle, Language: a.language,
		Activity: a.activity(), Usage: a.chat.displayTokens(), Status: a.status,
		Confirm: a.permissionConfirm,
		Rows:    []view.SettingRow{{Label: a.language.Text("Permission mode"), Values: []string{a.language.Text("Default"), "All Mighty"}, Selected: a.choice}},
		Details: []string{a.language.Text(description)},
	}, w, h)
}
