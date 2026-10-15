// Purpose: Verify review rejection reasons appear in the terminal while human approval remains independent.
// Pattern: Terminal event and key workflow tests.
// Usage: go test ./internal/tui/session -run TestConclave.
// Related: internal/tui/session/events.go, internal/tui/session/render.go, internal/tui/transcript/conclave.go.
package session

import (
	"github.com/gdamore/tcell/v3"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/tui/transcript"
	"strings"
	"testing"
)

func TestConclaveTerminalOutcomeAndApproval(t *testing.T) {
	a := fixture(t)
	event := run.Event{Kind: "conclave", Call: &tool.Call{ID: "c", Name: "edit_file"}, Conclave: &conclave.Result{Verdict: conclave.Verdict{Decision: "reject", Findings: []conclave.Finding{{Path: "a", Side: "request", Reason: "specific reviewer reason"}}}}}
	a.receive(notice{Event: &event})
	if len(a.output.Blocks) != 1 || !strings.Contains(a.output.Blocks[0].Text, transcript.ReviewRejectedMessage) {
		t.Fatal("rejection notice missing")
	}
	a.render()
	var visible strings.Builder
	for y := 0; y < 24; y++ {
		visible.WriteString(screenLine(a.screen, y, 80))
	}
	if !strings.Contains(visible.String(), "edit_file · Conclave Fail") || !strings.Contains(visible.String(), "specific reviewer reason") {
		t.Fatal(visible.String())
	}
	event.Conclave.Passed = true
	a.receive(notice{Event: &event})
	if a.status != "edit_file · Conclave Pass" || len(a.output.Blocks) != 1 {
		t.Fatal("unexpected pass rendering")
	}
	a.decision = make(chan bool, 1)
	a.receive(notice{Approval: &tool.Approval{Name: "Editing", Target: "a", Required: true, ConclavePassed: true, Preview: "-old\n+new"}})
	a.render()
	if !strings.Contains(screenLine(a.screen, 0, 80), "Conclave Pass") {
		t.Fatal("review pass missing from approval")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if <-a.decision {
		t.Fatal("review replaced user rejection")
	}
}
