// Purpose: Verify completion measurements appear after the last response with centered, copyable muted text.
// Pattern: Virtual terminal contract tests.
// Usage: go test ./internal/tui/session -run TestRunSummary.
// Related: internal/tui/session/runtime_events.go, internal/tui/session/worker_result.go, internal/tui/transcript/run_summary.go.
package session

import (
	"context"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/tui/locale"
	"github.com/rivo/uniseg"
)

func TestRunSummaryPlacementStyleAndCopy(t *testing.T) {
	a := fixture(t)
	a.language = locale.English
	a.busy, a.agentActive = true, true
	a.tokenTotal = modelapi.TokenUsage{Input: 9999, Output: 8888}
	a.receiveRuntime(&run.Event{Kind: "response", Text: "Finished the requested work."})
	a.receiveRuntime(&run.Event{Statistics: &run.Statistics{ElapsedMillis: 3723000, GenerationMillis: 4000, Usage: &modelapi.TokenUsage{Input: 120, Output: 20}}})
	if len(a.output.Blocks) != 1 {
		t.Fatal("completion line displayed before worker finished")
	}
	a.receive(notice{Done: true, RunStatus: run.Answered})
	if len(a.output.Blocks) != 2 || a.runSummary != nil {
		t.Fatal("completion line missing, duplicated or retained")
	}
	want := "Worked for 01:02:03 | INPUT 120 | OUTPUT 20 | TPS 5.0"
	for _, width := range []int{100, 36, 120} {
		a.screen.SetSize(width, 24)
		a.render()
		for _, line := range a.output.Lines(width - 1) {
			if line.ID != a.output.Blocks[1].ID || strings.TrimSpace(line.Text) == "" {
				continue
			}
			text := line.Copy.Row.Text
			padding := (width - 1 - uniseg.StringWidth(text)) / 2
			if line.Text != strings.Repeat(" ", padding)+text {
				t.Fatalf("width %d: summary is not centered: %q", width, line.Text)
			}
		}
	}
	y := -1
	for row := 1; row < a.inputRegion.y; row++ {
		if strings.Contains(screenLine(a.screen, row, 120), want) {
			y = row
			break
		}
	}
	x := (119 - len(want)) / 2
	if y <= 1 {
		t.Fatal("completion line not below the response")
	}
	_, style, _ := a.screen.Get(x, y)
	if style.GetAttributes()&tcell.AttrDim == 0 || style.GetBackground() != tcell.ColorDefault {
		t.Fatal("completion line is not muted on the terminal background")
	}
	drag(a, x, y, x+len(want)-1, y)
	if got := a.textSelection.Text(); got != want {
		t.Fatalf("centered summary copy=%q", got)
	}
	if a.output.Blocks[0].Centered || a.output.Blocks[0].Thinking {
		t.Fatal("answer inherited completion-line styling")
	}
}

func TestRunSummaryFollowsInterruptedDraft(t *testing.T) {
	a := fixture(t)
	a.busy, a.agentActive, a.stopRequested = true, true, true
	a.receiveRuntime(&run.Event{Kind: "stream", Text: "partial answer", Thinking: "partial reasoning"})
	a.receiveRuntime(&run.Event{Statistics: &run.Statistics{ElapsedMillis: 1000}})
	a.receive(notice{Done: true, RunStatus: run.Interrupted, Err: context.Canceled})
	if len(a.output.Blocks) != 3 || a.output.Blocks[1].Markdown == false || !a.output.Blocks[2].Centered || a.runSummary != nil {
		t.Fatalf("interrupted response ordering: %+v", a.output.Blocks)
	}
	if strings.TrimSpace(a.output.Blocks[1].Text) != "partial answer" || !strings.Contains(a.output.Blocks[2].Text, "입력 — | 출력 — | 초당 토큰 —") {
		t.Fatal("partial response or unreported usage lost")
	}
	a.receive(notice{Done: true})
	if len(a.output.Blocks) != 3 {
		t.Fatal("later worker reused the previous summary")
	}
}
