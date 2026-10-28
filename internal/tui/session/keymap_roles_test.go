// Purpose: Verify contextual help advertises only actions supported by the current focus and worker state.
// Pattern: Keyboard contract tests.
// Usage: go test ./internal/tui/session -run TestKeymapRoles.
// Related: internal/tui/session/keymap.go, internal/tui/session/keys.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/tui/preview"
	"strings"
	"testing"
)

func TestKeymapRolesInputOmitsCursorHint(t *testing.T) {
	for _, page := range []string{"chat", "files"} {
		t.Run(page, func(t *testing.T) {
			a := fixture(t)
			a.page, a.browser.Input = page, true
			a.screen.SetSize(240, 24)
			a.editor.Set("first\nsecond")
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
	a := fixture(t)
	typeInput(a, "!")
	if labels := keymapLabels(t, a); labels["↑↓"] != "선택" {
		t.Fatal(labels)
	}
}

func TestKeymapRolesFolderAndPreview(t *testing.T) {
	a := fixture(t)
	a.page = "files"
	a.browser.Open(".", []fileview.Entry{{Name: "sub", Directory: true}}, false)
	labels := map[string]string{}
	for _, s := range a.shortcuts() {
		labels[s.Key] = s.Label
	}
	if labels["Tab"] != "" || labels["Enter"] != "" || labels["Shift+Tab"] == "" {
		t.Fatal(labels)
	}
	a.filePreview = &preview.State{Full: false}
	labels = map[string]string{}
	for _, s := range a.shortcuts() {
		labels[s.Key] = s.Label
	}
	if labels["Tab"] != "" || labels["Enter"] == "" || labels["Esc"] != "닫기" {
		t.Fatal(labels)
	}
	a.filePreview.Full = true
	labels = map[string]string{}
	for _, s := range a.shortcuts() {
		labels[s.Key] = s.Label
	}
	if labels["Enter"] != "" || labels["Tab"] == "" {
		t.Fatal(labels)
	}
}

func TestKeymapRolesBusyInputAndCommands(t *testing.T) {
	a := fixture(t)
	a.worker = workerAgent
	a.page = "files"
	a.browser.Input = true
	labels := map[string]string{}
	for _, s := range a.shortcuts() {
		labels[s.Key] = s.Label
	}
	if labels["Enter"] != "방향 수정 확인" || labels["Tab"] != "예약 확인" || labels["Shift+Enter"] != "줄바꿈" {
		t.Fatal(labels)
	}
	a.page = "chat"
	typeInput(a, "!mo")
	labels = map[string]string{}
	for _, s := range a.shortcuts() {
		labels[s.Key] = s.Label
	}
	if labels["Tab"] != "삽입" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if a.editor.String() != "!model" || len(a.queued.Messages) != 0 || a.page != "chat" {
		t.Fatal("command Tab did not match keymap")
	}
}

func TestKeymapRolesNoMatchesAndBusyClose(t *testing.T) {
	a := fixture(t)
	typeInput(a, "!unknown")
	for _, s := range a.shortcuts() {
		if s.Key == "Enter" || s.Key == "Tab" || s.Key == "↑↓" {
			t.Fatalf("no-match action advertised: %+v", s)
		}
	}
	a.closeCommandHints()
	a.page = "files"
	a.browser.Input = true
	a.worker = workerSetup
	if labels := keymapLabels(t, a); labels["Esc"] != "닫기" || labels[":q"] != "" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "chat" || !a.worker.active() {
		t.Fatal("return hint did not match action")
	}
}
