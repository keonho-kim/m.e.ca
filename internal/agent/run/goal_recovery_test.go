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

func TestGoalAnalysisFailureStillAnswersGreeting(t *testing.T) {
	g := &generator{replies: []string{"Hello!", "Hello!", "Hello!", "DONE\nHello! How can I help?"}}
	e, j := &executor{}, &journal{}
	responses := 0
	result, err := fixture(g, e, j).RunGoal(context.Background(), "Hello", approve, func(event Event) error {
		if event.Kind == "response" {
			responses++
		}
		return nil
	})
	if err != nil || result.Status != Answered || result.Text != "Hello! How can I help?" || responses != 1 || e.prepared != 0 {
		t.Fatalf("result=%+v responses=%d prepared=%d err=%v", result, responses, e.prepared, err)
	}
	fallback, completed, users := false, false, 0
	for _, message := range j.messages {
		if message.Role == "user" {
			users++
		}
	}
	for _, event := range j.events {
		if event.Kind == "goal_analysis_fallback" && event.Internal {
			fallback = true
		}
		if event.Goal != nil && event.Goal.Status == "completed" {
			completed = event.Goal.Request == "Hello" && len(event.Goal.Tasks) == 1 && event.Goal.Tasks[0].Kind == "answer"
		}
	}
	if !fallback || !completed || users != 1 {
		t.Fatalf("fallback=%t completed=%t user messages=%d", fallback, completed, users)
	}
}

func TestGoalAnalysisFallbackDoesNotCompleteUnverifiedChanges(t *testing.T) {
	g := &generator{replies: []string{"bad", "bad", "bad", `{"tool":"write_file"}`, "a", "new", "DONE\nAll fixed", "WAIT\nVerification is still needed"}}
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
	g := &generator{replies: []string{"bad", "bad", "bad", "DONE\nHello"}}
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
	g := &generator{replies: []string{"partial"}, cancelAt: 1, cancel: cancel}
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
	g := &generator{replies: []string{"bad", "bad", "bad", `{"tool":"write_file"}`, "a", "new", `{"tool":"execute_command"}`, "test a", "DONE\nVerified"}, purposes: []string{"check"}}
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
