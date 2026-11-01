// Purpose: Verify LAT output streams between rules and its shared header indicator cleans up.
// Pattern: Virtual terminal behavior regressions.
// Usage: go test ./internal/tui/session -run TestLATOutput.
// Related: internal/tui/session/lat_output.go, internal/tui/session/activity.go.
package session

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestLATOutputStreamsThoughtBodyAndToolsBetweenRules(t *testing.T) {
	a := screenFixture(t)
	a.work.state, a.language = workerAgent, locale.English
	a.chat.output.Append("Main before", false)
	scope := &runrecord.LATRun{ID: "phase_plan", ParentRunID: "main_run", Phase: "plan"}
	send := func(event runrecord.Event) {
		event.Internal, event.LAT = true, scope
		a.receiveRuntime(&event)
	}
	send(runrecord.Event{Kind: "lat_phase_started"})
	send(runrecord.Event{Kind: "stream", Text: "**Inspecting sources**", Thinking: "Check requirements"})
	layout := a.chatLayout(74)
	shown := latScreenText(layout)
	if strings.Count(shown, "Look At This") != 1 || !strings.Contains(shown, "Inspecting sources") || !strings.Contains(shown, "Check requirements") || strings.HasSuffix(shown, strings.Repeat("-", 74)+"\n") {
		t.Fatalf("live LAT group missing or closed early: %q", shown)
	}
	send(runrecord.Event{Kind: "thinking", Thinking: "Check requirements"})
	send(runrecord.Event{Kind: "progress", Text: "**Inspecting sources**"})
	call := &toolapi.Call{ID: "read_1", Name: "execute_command", Arguments: json.RawMessage(`{"command":"cat README.md","workdir":"."}`)}
	send(runrecord.Event{Kind: "tool_started", Call: call, CallID: call.ID})
	if got := latScreenText(a.chatLayout(74)); !strings.Contains(got, "cat README.md") {
		t.Fatal("running LAT tool hidden", got)
	}
	send(runrecord.Event{Kind: "tool_succeeded", Call: call, CallID: call.ID, Output: json.RawMessage(`{"Stdout":"Source body","Status":"exited","ExitCode":0}`)})
	send(runrecord.Event{Kind: "response", Text: "Plan ready"})
	send(runrecord.Event{Kind: "lat_phase_finished"})
	a.receiveRuntime(&runrecord.Event{Kind: "lat_prepared"})
	a.receiveRuntime(&runrecord.Event{Kind: "stream", Text: "Main next"})
	shown = latScreenText(a.chatLayout(74))
	for _, text := range []string{"Check requirements", "Inspecting sources", "cat README.md", "Source body", "Plan ready"} {
		if strings.Count(shown, text) != 1 {
			t.Fatalf("missing or repeated %q: %s", text, shown)
		}
	}
	if !strings.Contains(shown, strings.Repeat("-", 74)+"\nMain next") || a.chat.latOutput.group.ID != "" || a.work.indicatorTimer != nil {
		t.Fatalf("LAT group leaked into main output: %s", shown)
	}
	if !a.chat.output.ToggleDetails() || strings.Contains(latScreenText(a.chatLayout(74)), "Source body") {
		t.Fatal("LAT tool output ignores normal detail toggle")
	}
}

func TestLATOutputHeaderAnimationSharesTimerAndStops(t *testing.T) {
	for _, width := range []int{100, 80, 40} {
		a := screenFixture(t)
		a.work.state = workerAgent
		a.screen.SetSize(width, 24)
		a.receiveRuntime(&runrecord.Event{Kind: "lat_preparing", Text: "plan"})
		for _, dots := range []int{1, 2, 3, 1} {
			a.render()
			if header := screenLine(a.screen, 0, width); !strings.Contains(header, "Running Look At This "+strings.Repeat(".", dots)) {
				t.Fatalf("width=%d dots=%d header=%q", width, dots, header)
			}
			a.work.advanceIndicator()
		}
		timer := a.work.indicatorTimer
		a.startCompression()
		if a.work.indicatorTimer != timer || !a.activity().Compressing {
			t.Fatal("overlapping LAT/compaction duplicated the indicator timer")
		}
		a.stopCompression()
		if a.work.indicatorTimer != timer || !a.activity().LAT {
			t.Fatal("compaction completion stopped the active LAT indicator")
		}
		a.receiveRuntime(&runrecord.Event{Kind: "lat_prepared"})
		if a.work.indicatorTimer != nil || a.activity().LAT || a.activity().Compressing {
			t.Fatal("preparation retained its timer")
		}
	}
}

func TestLATOutputReviewAndCancellationKeepSingleFeedbackAndPartialText(t *testing.T) {
	a := screenFixture(t)
	a.work.state = workerAgent
	scope := &runrecord.LATRun{ID: "review_1", ParentRunID: "main_run", Phase: "review"}
	send := func(kind, text string) {
		a.receiveRuntime(&runrecord.Event{Kind: kind, Text: text, Internal: true, LAT: scope})
	}
	send("lat_phase_started", "")
	send("stream", "NO_FIND")
	if strings.Contains(latScreenText(a.chatLayout(74)), "NO_FIND") {
		t.Fatal("machine verdict leaked while streaming")
	}
	send("stream", "- Missing endpoint")
	send("response", "- Missing endpoint")
	send("lat_phase_finished", "")
	a.receiveRuntime(&runrecord.Event{Kind: "lat_feedback", RunID: "main_run", Text: "- Missing endpoint"})
	if shown := latScreenText(a.chatLayout(74)); strings.Count(shown, "Missing endpoint") != 1 || a.chat.latOutput.group.ID != "" {
		t.Fatal("review feedback duplicated or left its group active", shown)
	}
	send("lat_phase_started", "")
	a.receiveRuntime(&runrecord.Event{Kind: "stream", Text: "Partial review", Thinking: "Partial thought", LAT: scope, Internal: true})
	a.finishWorker(notice{Done: true, Err: context.Canceled})
	shown := latScreenText(a.chatLayout(74))
	if strings.Count(shown, "Partial review") != 1 || strings.Count(shown, "Partial thought") != 1 || a.work.indicatorTimer != nil || a.activity().LAT {
		t.Fatal("cancelled LAT lost partial output or retained its indicator", shown)
	}
}

func latScreenText(layout *viewport.Transcript) string {
	var text strings.Builder
	for _, row := range layout.Window(0, layout.Count()) {
		text.WriteString(row.Text + "\n")
	}
	return text.String()
}
