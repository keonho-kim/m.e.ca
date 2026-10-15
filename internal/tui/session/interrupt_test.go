// Purpose: Verify Esc interrupts only active work and retains the session draft and partial answer.
// Pattern: Terminal lifecycle regression tests.
// Usage: go test ./internal/tui/session -run TestEscape.
// Related: internal/tui/session/keys.go, internal/tui/session/worker_result.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/agent/run"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestEscapeStopsTaskWithoutExiting(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.busy, a.agentActive = true, true
	a.taskCancel = cancel
	a.editor.Set("다음 지시")
	a.receive(notice{Event: &run.Event{Kind: "stream", Text: "부분 응답"}})
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.exit || ctx.Err() != context.Canceled || !a.busy || a.editor.String() != "다음 지시" {
		t.Fatal("Esc lost state or failed to cancel")
	}
	a.receive(notice{Approval: &toolapi.Approval{Name: "Editing"}})
	if a.page == "approval" {
		t.Fatal("late approval reopened after cancellation")
	}
	a.receive(notice{Done: true, Err: context.Canceled})
	if a.busy || a.agentActive || strings.Contains(a.status, "context canceled") || a.ctx.Err() != nil {
		t.Fatal("session did not return to idle")
	}
	if len(a.output.Blocks) != 1 || !strings.Contains(a.output.Blocks[0].Text, "부분 응답") || !a.output.Blocks[0].Markdown {
		t.Fatal("partial answer lost")
	}
}
func TestCompressionIndicatorLifecycle(t *testing.T) {
	a := fixture(t)
	a.busy, a.agentActive = true, true
	a.receive(notice{Event: &run.Event{Kind: "compaction_started"}})
	defer a.stopCompression()
	a.editor.Set("draft")
	a.render()
	header := screenLine(a.screen, 0, 80)
	if !strings.HasPrefix(header, "STATUS RUNNING 00 : 00 : 00") || !strings.Contains(header, "Compress Memory .") || !strings.HasSuffix(header, "INPUT 0 · OUTPUT 0 · TPS —") {
		t.Fatalf("header alignment: %q", header)
	}
	if strings.TrimSpace(screenLine(a.screen, 21, 80)) != "> draft" {
		t.Fatal("status moved into composer")
	}
	if a.compressionTicks == nil {
		t.Fatal("compression timer missing")
	}
	for want := 1; want <= 3; want++ {
		if !strings.Contains(a.activity().Label(), "Compress Memory "+strings.Repeat(".", want)) {
			t.Fatal(a.activity().Label())
		}
		a.advanceCompression()
	}
	a.receive(notice{Event: &run.Event{Kind: "compaction_finished"}})
	if a.compressionTicks != nil || a.activity().Compressing {
		t.Fatal("compression timer leaked")
	}
	a.receive(notice{Done: true})
	a.render()
	if !strings.HasPrefix(screenLine(a.screen, 0, 80), "STATUS IDLE") || !strings.HasSuffix(screenLine(a.screen, 0, 80), "INPUT 0 · OUTPUT 0 · TPS —") {
		t.Fatal("idle status missing")
	}
}
