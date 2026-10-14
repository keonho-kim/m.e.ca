// Purpose: Verify Goal parsing tolerates layout whitespace without relaxing task bounds or kinds.
// Pattern: Parser contract tests.
// Usage: go test ./internal/agent/goal -run TestAnalysis.
// Related: internal/agent/goal/state.go.
package goal

import (
	"strings"
	"testing"
)

func TestAnalysisAllowsBlankLinesAndIndentation(t *testing.T) {
	text := "\r\n  GOAL: Reply to the greeting\r\n\r\n  TASK answer: Greet the user\r\n\r\n"
	state, err := Parse(text, "g", "Hello")
	if err != nil || len(state.Tasks) != 1 || state.Tasks[0].Kind != "answer" {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if _, err := Parse("GOAL: work\n"+strings.Repeat("\nTASK work: next\n", 9), "g", "work"); err == nil {
		t.Fatal("whitespace handling bypassed task bound")
	}
}

func TestAnalysisFallbackPreservesFullRequestAndWorkRequirement(t *testing.T) {
	request := strings.Repeat("request ", 1000)
	for _, status := range []string{"answered", "verified"} {
		state, err := Unsplit("g", request)
		if err != nil || state.Request != request || len(state.Tasks) != 1 || state.Tasks[0].Kind != "work" {
			t.Fatalf("state=%+v err=%v", state, err)
		}
		if err := state.Start(0); err != nil {
			t.Fatal(err)
		}
		evidence, expected := "", "stopped"
		if status == "verified" {
			evidence, expected = "observed check", "completed"
		}
		if err := state.Finish(0, status, "response", evidence); err != nil || state.Status != expected || state.Tasks[0].Kind != "work" {
			t.Fatalf("state=%+v err=%v", state, err)
		}
	}
	if _, err := Parse("GOAL: shortcut\nTASK request: skip classification", "g", "work"); err == nil {
		t.Fatal("unsupported request kind accepted")
	}
}
