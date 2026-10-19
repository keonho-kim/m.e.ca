// Purpose: Verify contextual help advertises only actions supported by the current focus and worker state.
// Pattern: Keyboard contract tests.
// Usage: go test ./internal/tui/session -run TestKeymapRoles.
// Related: internal/tui/session/keymap.go, internal/tui/session/keys.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/tools/local"
	"github.com/keonho-kim/v-code/internal/tui/browser"
	"testing"
)

func TestKeymapRolesFolderAndPreview(t *testing.T) {
	a := fixture(t)
	a.page = "files"
	a.browser.Open(".", []local.Entry{{Name: "sub", Directory: true}}, false)
	labels := map[string]string{}
	for _, s := range a.shortcuts() {
		labels[s.Key] = s.Label
	}
	if labels["Tab"] != "" || labels["Enter"] != "" || labels["Shift+Tab"] == "" {
		t.Fatal(labels)
	}
	a.filePreview = &browser.Preview{Full: false}
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
	a.busy, a.agentActive = true, true
	a.page = "files"
	a.browser.Input = true
	labels := map[string]string{}
	for _, s := range a.shortcuts() {
		labels[s.Key] = s.Label
	}
	if labels["Enter"] != "방향 수정" || labels["Tab"] != "후속 예약" || labels["Shift+Enter"] != "줄바꿈" {
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
	if a.editor.String() != "!model" || a.pending != "" || a.page != "chat" {
		t.Fatal("command Tab did not match keymap")
	}
}

func TestKeymapRolesNoMatchesAndBusyReturn(t *testing.T) {
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
	a.busy = true
	a.editor.Set(":q")
	found := false
	for _, s := range a.shortcuts() {
		if s.Key == "Enter" && s.Label == "복귀" {
			found = true
		}
	}
	if !found {
		t.Fatal("busy browser return missing")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "chat" || !a.busy {
		t.Fatal("return hint did not match action")
	}
}
