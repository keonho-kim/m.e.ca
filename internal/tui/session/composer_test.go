// Purpose: Verify a fixed bottom composer, transient stop notices and Shift+Enter input.
// Pattern: Terminal lifecycle regression tests.
// Usage: go test ./internal/tui/session -run TestFixedComposer.
// Related: internal/tui/session/render_chat.go, internal/tui/session/keys.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"testing"
)

func TestFixedComposerAndStopNotice(t *testing.T) {
	a := fixture(t)
	a.editor.Set("draft")
	for _, response := range []string{"", "working", strings.Repeat("response\n", 40)} {
		a.receive(notice{Event: &run.Event{Kind: "stream", Text: response}})
		a.render()
		if strings.TrimSpace(screenLine(a.screen, 21, 80)) != "> draft" {
			t.Fatal("composer moved with transcript")
		}
	}
	_, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.busy, a.agentActive, a.taskCancel = true, true, cancel
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.receive(notice{Done: true, Err: context.Canceled})
	a.render()
	if !strings.Contains(screenLine(a.screen, 20, 80), "중지했습니다") {
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
	if strings.Contains(screenLine(a.screen, 20, 80), "중지했습니다") {
		t.Fatal("stop notice reappeared")
	}
}

func TestFixedComposerShiftEnter(t *testing.T) {
	a := fixture(t)
	typeInput(a, "!quit")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModShift))
	if a.exit || a.editor.String() != "!quit\n" {
		t.Fatal("Shift+Enter executed command")
	}
	a.render()
	if strings.TrimSpace(screenLine(a.screen, 20, 80)) != "> !quit" {
		t.Fatal("multiline composer did not grow upward")
	}
}

func TestFixedSystemScopeOnSessionRestore(t *testing.T) {
	a := fixture(t)
	if err := a.opts.Store.SaveSettings(a.ctx, policy.Settings{Scope: policy.Directory, Permission: policy.Approve}); err != nil {
		t.Fatal(err)
	}
	if err := a.restore(); err != nil {
		t.Fatal(err)
	}
	if a.settings.Scope != policy.System || a.settings.Permission != policy.Approve {
		t.Fatal("saved scope changed fixed policy or approval")
	}
	typeInput(a, "!scope")
	a.submit()
	if a.page == "scope" || a.settings.Scope != policy.System {
		t.Fatal("scope selection still available")
	}
}

func TestGlobalDefaultPermissionIsAutomatic(t *testing.T) {
	a := fixture(t)
	if a.settings.Permission != policy.AllowAll || a.opts.Config.Defaults.Permission != policy.AllowAll {
		t.Fatal("global default not inherited")
	}
	typeInput(a, "!permission")
	a.submit()
	if a.page != "permission" || a.choices[a.choice] != "allow-all" {
		t.Fatal("menu did not select current permission")
	}
}
