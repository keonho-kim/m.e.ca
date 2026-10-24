// Purpose: Verify interruption preserves work and compression activity stays visible across terminal widths.
// Pattern: Terminal lifecycle regression tests.
// Usage: go test ./internal/tui/session -run 'TestEscape|TestCompressionIndicator'.
// Related: internal/tui/session/keys.go, internal/tui/session/worker_result.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/meca/internal/agent/run"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"strings"
	"testing"
)

func TestEscapeStopsTaskWithoutExiting(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	a.worker = workerAgent
	a.taskCancel = cancel
	a.editor.Set("다음 지시")
	a.receive(notice{Event: &run.Event{Kind: "stream", Text: "부분 응답"}})
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if a.exit || ctx.Err() != context.Canceled || !a.worker.active() || a.editor.String() != "다음 지시" {
		t.Fatal("Esc lost state or failed to cancel")
	}
	a.receive(notice{Approval: &toolapi.Approval{Name: "Editing"}})
	if a.page == "approval" {
		t.Fatal("late approval reopened after cancellation")
	}
	a.receive(notice{Done: true, Err: context.Canceled})
	if a.worker.active() || a.worker.agent() || strings.Contains(a.status, "context canceled") || a.ctx.Err() != nil {
		t.Fatal("session did not return to idle")
	}
	if len(a.output.Blocks) != 1 || !strings.Contains(a.output.Blocks[0].Text, "부분 응답") || !a.output.Blocks[0].Markdown {
		t.Fatal("partial answer lost")
	}
}

func TestCompressionIndicatorSurvivesNarrowHeader(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		for _, width := range []int{80, 60, 40} {
			a := fixture(t)
			a.language = lang
			a.screen.SetSize(width, 24)
			a.worker = workerAgent
			a.tokenTotal.Input, a.tokenTotal.Output = 1234567890, 9876543210
			a.receiveRuntime(&run.Event{Kind: "compaction_started"})
			a.advanceCompression()
			a.render()
			header := screenLine(a.screen, 0, width)
			if !strings.Contains(header, lang.Text("Compress Memory")+" ..") {
				t.Fatalf("language=%v width=%d: %q", lang, width, header)
			}
			a.stopCompression()
		}
	}
}
func TestCompressionIndicatorLifecycle(t *testing.T) {
	a := fixture(t)
	a.worker = workerAgent
	a.receive(notice{Event: &run.Event{Kind: "compaction_started"}})
	defer a.stopCompression()
	a.editor.Set("draft")
	a.render()
	header := screenLine(a.screen, 0, 80)
	if !strings.HasPrefix(header, "상태 작업 중 00 : 00 : 00") || !strings.Contains(header, "메모리 압축 중 .") || !strings.HasSuffix(header, "입력 0 · 출력 0 · 초당 토큰 —") {
		t.Fatalf("header alignment: %q", header)
	}
	if strings.TrimSpace(screenLine(a.screen, a.inputRegion.y, 80)) != "> draft" {
		t.Fatal("status moved into composer")
	}
	if a.compressionTicks == nil {
		t.Fatal("compression timer missing")
	}
	for want := 1; want <= 3; want++ {
		if !strings.Contains(a.activity().Label(), "메모리 압축 중 "+strings.Repeat(".", want)) {
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
	if !strings.HasPrefix(screenLine(a.screen, 0, 80), "상태 대기") || !strings.HasSuffix(screenLine(a.screen, 0, 80), "입력 0 · 출력 0 · 초당 토큰 —") {
		t.Fatal("idle status missing")
	}
}
