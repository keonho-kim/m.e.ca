// Purpose: Verify key hints match executable actions across active input and foreground states.
// Pattern: Virtual terminal contract tests.
// Usage: go test ./internal/tui/session -run TestKeymapActions.
// Related: internal/tui/session/keymap.go, internal/tui/session/keys.go, internal/tui/session/input_region.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"strings"
	"testing"
)

func keymapLabels(t *testing.T, a *app) map[string]string {
	t.Helper()
	labels := map[string]string{}
	for _, shortcut := range a.shortcuts() {
		if _, duplicate := labels[shortcut.Key]; duplicate {
			t.Fatalf("duplicate key hint: %s", shortcut.Key)
		}
		labels[shortcut.Key] = shortcut.Label
	}
	return labels
}

func TestKeymapActionsSteeringAndFollowup(t *testing.T) {
	for _, page := range []string{"chat", "files"} {
		t.Run(page, func(t *testing.T) {
			a := fixture(t)
			a.page, a.browser.Input = page, true
			a.busy, a.agentActive = true, true
			a.steering = make(chan string, 1)
			a.editor.Set("Run this after finishing")
			labels := keymapLabels(t, a)
			enter := "방향 수정"
			tab := "후속 예약"
			if page == "files" {
				enter = "전송 확인"
				tab = "예약 확인"
			}
			if labels["Enter"] != enter || labels["Tab"] != tab {
				t.Fatal(labels)
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
			if page == "files" {
				a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
				a.page, a.browser.Input = "files", true
			}
			if a.pending != "Run this after finishing" {
				t.Fatal("follow-up hint did not match action")
			}
			a.editor.Set("Change direction now")
			labels = keymapLabels(t, a)
			if labels["Tab"] != "" || labels["Enter"] != enter {
				t.Fatal(labels)
			}
			a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
			if page == "files" {
				a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
			}
			select {
			case text := <-a.steering:
				if text != "Change direction now" {
					t.Fatal(text)
				}
			default:
				t.Fatal("steering hint did not match action")
			}
		})
	}
}

func TestKeymapActionsStopAndBlockedCommands(t *testing.T) {
	for _, text := range []string{"continue", "!model", "!unknown", "!model extra"} {
		a := fixture(t)
		a.busy, a.agentActive = true, true
		a.stopRequested = text == "continue"
		a.editor.Set(text)
		labels := keymapLabels(t, a)
		if labels["Enter"] != "" || labels["Tab"] != "" || a.stopRequested && labels["Esc"] != "" {
			t.Fatalf("%s: %v", text, labels)
		}
	}
}

func TestKeymapActionsFieldReturn(t *testing.T) {
	for _, page := range []string{"model"} {
		a := fixture(t)
		a.page = page
		a.openModel()
		a.menuInput.Set(":q")
		labels := keymapLabels(t, a)
		if labels["Enter"] != "복귀" || labels["Tab"] != "" || labels["↑↓"] != "" {
			t.Fatalf("%s: %v", page, labels)
		}
		a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
		if a.page != "chat" {
			t.Fatal("field return hint did not match action")
		}
	}
}

func TestKeymapActionsBusyQuitConfirmation(t *testing.T) {
	a := fixture(t)
	a.busy, a.agentActive = true, true
	a.editor.Set(":q")
	if labels := keymapLabels(t, a); labels["Enter"] != "종료 확인" || labels["Tab"] != "" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "quit" || a.exit {
		t.Fatal("quit confirmation hint did not match action")
	}
}

func TestKeymapActionsCloseFileWhileLoading(t *testing.T) {
	a := fixture(t)
	a.page, a.directoryLoading = "files", true
	labels := keymapLabels(t, a)
	if labels["Esc"] != "닫기" || labels["Enter"] != "" || labels[":q"] != "" {
		t.Fatal(labels)
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.page != "chat" || a.directoryLoading {
		t.Fatal("Esc should cancel lookup and close the browser")
	}
}

func TestKeymapActionsDeleteSelection(t *testing.T) {
	for _, scope := range []string{"input", "cross-boundary", "stale"} {
		t.Run(scope, func(t *testing.T) {
			a := fixture(t)
			a.editor.Set("draft text")
			a.render()
			y := a.inputRegion.y
			endY := y
			if scope == "cross-boundary" {
				endY--
			}
			drag(a, 2, y, 6, endY)
			if scope == "stale" {
				a.editor.Set("new draft")
			}
			labels := keymapLabels(t, a)
			if got := labels["Backspace/Del"] != ""; got != (scope == "input") {
				t.Fatal(labels)
			}
			before := a.editor.String()
			a.keyEvent(tcell.NewEventKey(tcell.KeyDelete, "", 0))
			if changed := a.editor.String() != before; changed != (scope == "input") {
				t.Fatal("selection deletion hint did not match action")
			}
		})
	}
}

func TestKeymapActionsFooterOrder(t *testing.T) {
	a := fixture(t)
	a.screen.SetSize(160, 24)
	a.busy, a.agentActive = true, true
	a.editor.Set("direction")
	a.render()
	footer := screenLine(a.screen, 23, 160)
	keys := []string{"^C", "Esc", "Enter", "Tab"}
	last := -1
	for _, key := range keys {
		at := strings.Index(footer, key)
		if at <= last {
			t.Fatalf("footer order: %q", footer)
		}
		last = at
	}
}
