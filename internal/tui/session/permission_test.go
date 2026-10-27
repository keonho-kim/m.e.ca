// Purpose: Verify approval settings save, cancel and failure behavior with bilingual keymaps.
// Pattern: Virtual terminal settings contract tests.
// Usage: go test ./internal/tui/session -run TestPermission.
// Related: internal/tui/session/permission.go, internal/commands/settings/permission.go.
package session

import (
	"errors"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestPermissionSaveAndKeymap(t *testing.T) {
	for _, language := range []locale.Language{locale.Korean, locale.English} {
		a := fixture(t)
		a.language = language
		typeInput(a, "!permission")
		a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
		a.render()
		if a.page != "permission" || !strings.Contains(screenLine(a.screen, 2, 80), "All Mighty") || keymapLabels(t, a)["←→"] != language.Text("Select") {
			t.Fatal("permission screen or selector hint missing")
		}
		a.editor.Set("draft")
		a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
		a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
		if labels := keymapLabels(t, a); labels["y"] != language.Text("Save") || labels["←→"] != "" {
			t.Fatal("confirmation keymap did not replace the selector", labels)
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
		saved, err := config.Load(a.opts.State)
		if err != nil || a.page != "chat" || a.opts.Config.Permission != policy.PermissionAllMighty || saved.Permission != policy.PermissionAllMighty || a.editor.String() != "draft" {
			t.Fatalf("permission not saved or draft lost: mode=%v saved=%v error=%v", a.opts.Config.Permission, saved.Permission, err)
		}
		a.openPermission()
		if a.choice != 1 {
			t.Fatal("saved permission not selected")
		}
	}
}

func TestPermissionCancelFailureAndActiveTask(t *testing.T) {
	a := fixture(t)
	a.openPermission()
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "permission" || a.permissionConfirm || a.choice != 1 {
		t.Fatal("Esc did not return to the staged setting")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "n", 0))
	if a.page != "permission" || !a.permissionConfirm {
		t.Fatal("lowercase n dismissed confirmation")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "N", 0))
	if a.page != "chat" || a.opts.Config.Permission != policy.PermissionDefault {
		t.Fatal("discard applied a permission change")
	}
	a.opts.SaveConfig = func(config.Config) error { return errors.New("write failed") }
	a.openPermission()
	a.choice = 1
	a.savePermission()
	if a.page != "permission" || a.opts.Config.Permission != policy.PermissionDefault || !strings.Contains(a.status, "write failed") {
		t.Fatal("failed save applied the mode")
	}
	a.page, a.worker = "chat", workerAgent
	a.editor.Set("!permission")
	a.submit()
	if a.page != "chat" || !a.worker.active() || a.opts.Config.Permission != policy.PermissionDefault {
		t.Fatal("active task allowed its permission policy to change")
	}
}
