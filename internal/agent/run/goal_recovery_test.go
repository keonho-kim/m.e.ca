// Purpose: Verify malformed Goal analysis does not block conversation or bypass verification and failures.
// Pattern: Goal orchestration regression tests.
// Usage: go test ./internal/agent/run -run TestGoalAnalysis.
// Related: internal/agent/run/goal_model.go, internal/agent/run/goal.go, internal/agent/goal/state.go.
package run

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGoalAnalysisFallbackDoesNotCompleteUnverifiedChanges(t *testing.T) {
	g := &generator{replies: []string{`{"tool":"write_file"}`, "bad", "bad", "bad", `{"tool":"write_file"}`, "a", "new", "DONE\nAll fixed", "WAIT\nVerification is still needed"}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).RunGoal(context.Background(), "Update a and verify it", approve, emit)
	if err != nil || result.Status != NeedsInput || e.executed != 1 {
		t.Fatalf("result=%+v executed=%d err=%v", result, e.executed, err)
	}
	for _, event := range j.events {
		if event.Goal != nil && event.Goal.Status == "completed" {
			t.Fatal("unverified fallback Goal marked completed")
		}
	}
}

func TestGoalAnalysisFallbackSharesBudget(t *testing.T) {
	g := &generator{replies: []string{`{"tool":"write_file"}`, "bad", "bad", "bad", "DONE\nHello"}}
	r := fixture(g, &executor{}, &journal{})
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 3
	if _, err := r.RunGoal(context.Background(), "Hello", approve, emit); err == nil || !strings.Contains(err.Error(), "budget") || len(g.requests) != 3 {
		t.Fatalf("calls=%d err=%v", len(g.requests), err)
	}
}

func TestGoalAnalysisCancellationDoesNotFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := &generator{replies: []string{`{"tool":"read_file"}`, "partial"}, cancelAt: 2, cancel: cancel}
	j := &journal{}
	if _, err := fixture(g, &executor{}, j).RunGoal(ctx, "Hello", approve, emit); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	for _, event := range j.events {
		if event.Kind == "goal_analysis_fallback" {
			t.Fatal("canceled analysis continued")
		}
	}
}

func TestGoalAnalysisFallbackRecordsVerifiedWork(t *testing.T) {
	g := &generator{replies: []string{`{"tool":"write_file"}`, "bad", "bad", "bad", `{"tool":"write_file"}`, "a", "new", `{"tool":"execute_command"}`, "test a", "DONE\nVerified"}, purposes: []string{"check"}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).RunGoal(context.Background(), "Update a and verify it", approve, emit)
	if err != nil || result.Status != Verified || e.executed != 2 {
		t.Fatalf("result=%+v executed=%d err=%v", result, e.executed, err)
	}
	for _, event := range j.events {
		if event.Goal != nil && event.Goal.Status == "completed" && event.Goal.Tasks[0].Kind == "work" && event.Goal.Tasks[0].Evidence != "" {
			return
		}
	}
	t.Fatal("verified work was not recorded")
}
