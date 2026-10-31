// Purpose: Verify a fixed bottom composer, transient stop notices and Shift+Enter input.
// Pattern: Terminal lifecycle regression tests.
// Usage: go test ./internal/tui/session -run TestFixedComposer.
// Related: internal/tui/session/render_chat.go, internal/tui/session/keys.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/policy"
	"strings"
	"testing"
)

func TestFixedComposerAndStopNotice(t *testing.T) {
	a := screenFixture(t)
	a.editor.Set("draft")
	inputRow := -1
	for _, response := range []string{"", "working", strings.Repeat("response\n", 40)} {
		a.receive(notice{Event: &runrecord.Event{Kind: "stream", Text: response}})
		a.render()
		if inputRow < 0 {
			inputRow = a.inputRegion.y
		}
		if a.inputRegion.y != inputRow || strings.TrimSpace(screenLine(a.screen, inputRow, 80)) != "> draft" {
			t.Fatal("composer moved with transcript")
		}
	}
	_, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.worker, a.taskCancel = workerAgent, cancel
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.receive(notice{Done: true, Err: context.Canceled})
	a.render()
	if !strings.Contains(screenLine(a.screen, a.inputRegion.y-1, 80), "중지했습니다") {
		t.Fatal("stop notice not above input")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyCtrlU, "", 0))
	typeInput(a, "!model")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.render()
	for y := 0; y < 24; y++ {
		if strings.Contains(screenLine(a.screen, y, 80), "중지했습니다") {
			t.Fatal("stop notice followed menu")
		}
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.render()
	if strings.Contains(screenLine(a.screen, a.inputRegion.y-1, 80), "중지했습니다") {
		t.Fatal("stop notice reappeared")
	}
}

func TestFixedComposerShiftEnter(t *testing.T) {
	a := screenFixture(t)
	typeInput(a, "!quit")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModShift))
	if a.exit || a.editor.String() != "!quit\n" {
		t.Fatal("Shift+Enter executed command")
	}
	a.render()
	if strings.TrimSpace(screenLine(a.screen, a.inputRegion.y, 80)) != "> !quit" {
		t.Fatal("multiline composer did not grow upward")
	}
}

func TestFixedSystemScopeOnSessionRestore(t *testing.T) {
	a := fixture(t)
	a.settings.Scope = policy.Directory
	if err := a.restore(); err != nil {
		t.Fatal(err)
	}
	if a.settings.Scope != policy.System {
		t.Fatal("restoration changed fixed application scope")
	}
}

func TestPermissionCommandOpensSettings(t *testing.T) {
	a := screenFixture(t)
	typeInput(a, "!permission")
	a.submit()
	if a.page != "permission" || a.status != "" || len(a.commandHints) != 0 {
		t.Fatal("permission settings did not open")
	}
}
