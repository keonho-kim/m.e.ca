// Purpose: Verify contextual help advertises only actions supported by the current focus and worker state.
// Pattern: Keyboard contract tests.
// Usage: go test ./internal/tui/session -run TestKeymapRoles.
// Related: internal/tui/session/keymap.go, internal/tui/session/keys.go.
package session

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/tui/preview"
)

func TestKeymapRolesInputOmitsCursorHint(t *testing.T) {
	for _, page := range []string{"chat", "files"} {
		t.Run(page, func(t *testing.T) {
			a := screenFixture(t)
			a.page, a.files.browser.Input = page, true
			a.screen.SetSize(240, 24)
			a.compose.editor.Set("first\nsecond")
			a.render()
			labels := keymapLabels(t, a)
			if labels["↑↓"] != "" || labels["Ctrl/Cmd+↑↓"] != "입력 기록" {
				t.Fatal(labels)
			}
			footer := screenLine(a.screen, 23, 240)
			if strings.Contains(footer, "줄 이동") || !strings.Contains(footer, "Ctrl/Cmd+↑↓") {
				t.Fatalf("input footer: %q", footer)
			}
		})
	}
	a := screenFixture(t)
	typeInput(a, "!")
	if labels := keymapLabels(t, a); labels["↑↓"] != "선택" {
		t.Fatal(labels)
	}
}

func TestKeymapRolesFolderAndPreview(t *testing.T) {
	a := screenFixture(t)
	a.page = "files"
	a.files.browser.Open(".", []fileview.Entry{{Name: "sub", Directory: true}}, false)
	labels := keymapLabels(t, a)
	if labels["Tab"] != "" || labels["Enter"] != "" || labels["Shift+Tab"] == "" {
		t.Fatal(labels)
	}
	a.files.filePreview = &preview.State{Full: false}
	labels = keymapLabels(t, a)
	if labels["Tab"] != "" || labels["Enter"] == "" || labels["Esc"] != "닫기" {
		t.Fatal(labels)
	}
	a.files.filePreview.Full = true
	labels = keymapLabels(t, a)
	if labels["Enter"] != "" || labels["Tab"] == "" {
		t.Fatal(labels)
	}
}

func TestKeymapRolesBusyInputAndCommands(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerAgent
	a.page = "files"
	a.files.browser.Input = true
	labels := keymapLabels(t, a)
	if labels["Enter"] != "방향 수정 확인" || labels["Tab"] != "예약 확인" || labels["Shift+Enter"] != "줄바꿈" {
		t.Fatal(labels)
	}
	a.page = "chat"
	typeInput(a, "!mo")
	labels = keymapLabels(t, a)
	if labels["Tab"] != "삽입" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if a.compose.editor.String() != "!model" || len(a.compose.queued.Messages) != 0 || a.page != "chat" {
		t.Fatal("command Tab did not match keymap")
	}
}

func TestKeymapRolesNoMatchesAndBusyClose(t *testing.T) {
	a := screenFixture(t)
	typeInput(a, "!unknown")
	for _, s := range a.shortcuts() {
		if s.Key == "Enter" || s.Key == "Tab" || s.Key == "↑↓" {
			t.Fatalf("no-match action advertised: %+v", s)
		}
	}
	a.closeCommandHints()
	a.page = "files"
	a.files.browser.Input = true
	a.work.state = workerSetup
	if labels := keymapLabels(t, a); labels["Esc"] != "닫기" || labels[":q"] != "" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "chat" || !a.work.state.active() {
		t.Fatal("return hint did not match action")
	}
}
