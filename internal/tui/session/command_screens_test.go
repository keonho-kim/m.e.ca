// Purpose: Verify settings commands stage changes and file navigation closes without stopping active work.
// Pattern: Virtual terminal command contract tests.
// Usage: go test ./internal/tui/session -run 'TestLanguageScreen|TestFileScreen'.
// Related: internal/tui/session/language.go, internal/tui/session/files.go, internal/tui/session/keys.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/tools/local"
	"github.com/keonho-kim/meca/internal/tui/browser"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestLanguageScreenStagesAndConfirms(t *testing.T) {
	a := fixture(t)
	a.language = locale.English
	a.openLanguage()
	if labels := keymapLabels(t, a); labels["→"] != "Next value" || labels["←"] != "Previous value" || labels["↑↓"] != "" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyLeft, "", 0))
	if a.choice != 2 {
		t.Fatal("left did not select the previous language")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRight, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "language" || a.opts.Config.Language != config.LanguageAuto || !strings.Contains(modelScreen(a), "Save current settings?") {
		t.Fatal("language changed before save confirmation")
	}
	if labels := keymapLabels(t, a); labels["y"] != "Save" || labels["n"] != "Discard" || labels["Esc"] != "Keep editing" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "language" || a.choice != 1 {
		t.Fatal("confirmation Esc lost the language draft")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "n", 0))
	if a.page != "chat" || a.opts.Config.Language != config.LanguageAuto || a.language != locale.English {
		t.Fatal("n applied the language draft")
	}
}

func TestFileScreenEscClosesForegroundBeforeActiveTask(t *testing.T) {
	a := fixture(t)
	a.page = "files"
	a.browser.Open(".", []local.Entry{{Name: "README.md"}}, false)
	a.browser.Input = true
	a.browser.Selected = map[string]bool{"README.md": true}
	a.editor.Set("keep draft")
	a.filePreview = &browser.Preview{Full: true}
	a.worker = workerAgent
	cancelled := false
	a.taskCancel = func() { cancelled = true }
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.filePreview != nil || a.page != "files" || !a.browser.Input || cancelled {
		t.Fatal("Esc did not close only the file preview")
	}
	if label := keymapLabels(t, a)["Esc"]; label != "닫기" {
		t.Fatalf("input Escape hint=%q", label)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "chat" || cancelled || a.editor.String() != "keep draft" || !a.browser.Selected["README.md"] {
		t.Fatal("closing file browser cancelled active work or lost the draft")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if !cancelled {
		t.Fatal("chat Escape no longer stops the active task")
	}
}

func TestFileScreenEscCancelsLookupAndCloses(t *testing.T) {
	a := fixture(t)
	a.page, a.directoryLoading = "files", true
	cancelled := false
	a.directoryReads.running = func() { cancelled = true }
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if !cancelled || a.directoryLoading || a.page != "chat" {
		t.Fatal("Esc did not cancel lookup and return to chat in one step")
	}
}
