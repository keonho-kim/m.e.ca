// Purpose: Verify bounded plain Goal analysis and sequential evidence-gated Task transitions.
// Pattern: State-machine contract tests.
// Usage: go test ./internal/agent/goal.
// Related: internal/agent/goal/state.go.
package goal

import "testing"

func TestGoalSequentialCompletion(t *testing.T) {
	state, err := Parse("GOAL: fix and explain\nTASK work: fix parser\nTASK answer: explain change", "g", "fix and explain")
	if err != nil {
		t.Fatal(err)
	}
	if err = state.Start(1); err == nil {
		t.Fatal("skipped previous Task")
	}
	if err = state.Start(0); err != nil {
		t.Fatal(err)
	}
	if err = state.Finish(0, "verified", "fixed", "focused test passed"); err != nil {
		t.Fatal(err)
	}
	if err = state.Start(1); err != nil {
		t.Fatal(err)
	}
	if err = state.Finish(1, "answered", "explanation", ""); err != nil || state.Status != "completed" {
		t.Fatalf("%+v %v", state, err)
	}
	state.Tasks[0].Evidence = ""
	if state.Check() == nil {
		t.Fatal("unverified completed work accepted")
	}
}
func TestWorkCannotFinishWithoutVerification(t *testing.T) {
	state, err := Parse("GOAL: fix\nTASK work: fix", "g", "fix")
	if err != nil {
		t.Fatal(err)
	}
	if err = state.Start(0); err != nil {
		t.Fatal(err)
	}
	if err = state.Finish(0, "answered", "claimed done", ""); err != nil || state.Status != "stopped" {
		t.Fatalf("%+v %v", state, err)
	}
	for _, text := range []string{`{"goal":"fix"}`, "GOAL: fix\nTASK unknown: fix", "GOAL: fix\nTASK work: "} {
		if _, err := Parse(text, "g", "fix"); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
}
