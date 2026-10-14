// Purpose: Verify hidden Goal Tasks share budgets and publish only the final user report.
// Pattern: Goal orchestration integration tests.
// Usage: go test ./internal/agent/run -run TestGoal.
// Related: internal/agent/run/goal.go, internal/agent/run/goal_model.go.
package run

import (
	"context"
	"strings"
	"testing"
)

func TestGoalRunsSequentialTasksAndPublishesOneReply(t *testing.T) {
	g := &generator{purposes: []string{"check"}, replies: []string{`{"tool":"read_file"}`, "GOAL: inspect and explain\nTASK work: inspect parser\nTASK answer: explain result", `{"tool":"read_file"}`, "parser.txt", "DONE\ninspection report", "DONE\nexplanation report", "DONE\nfinal user response"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	responses := 0
	result, err := r.RunGoal(context.Background(), "inspect and explain", approve, func(event Event) error {
		if event.Kind == "response" {
			responses++
			if event.Text != "final user response" {
				t.Errorf("internal report leaked: %s", event.Text)
			}
		}
		return nil
	})
	if err != nil || result.Status != Verified || result.Text != "final user response" || responses != 1 || e.executed != 1 {
		t.Fatalf("%+v %v responses=%d", result, err, responses)
	}
	users, answers := 0, 0
	for _, m := range j.messages {
		if m.Role == "user" {
			users++
		}
		if m.Role == "assistant" && len(m.Calls) == 0 {
			answers++
		}
	}
	if users != 1 || answers != 1 {
		t.Fatalf("duplicated conversation: %d/%d", users, answers)
	}
	completed := false
	for _, event := range j.events {
		if event.Goal != nil && event.Goal.Status == "completed" {
			completed = true
			if len(event.Goal.Tasks) != 2 || event.Goal.Tasks[0].Evidence == "" {
				t.Fatal("Goal evidence missing")
			}
		}
	}
	if !completed {
		t.Fatal("Goal completion not recorded")
	}
}
func TestGoalTasksShareIterationBudget(t *testing.T) {
	g := &generator{replies: []string{`{"tool":"read_file"}`, "GOAL: two answers\nTASK answer: first\nTASK work: second", "DONE\nfirst"}}
	r := fixture(g, &executor{}, &journal{})
	r.Limits = DefaultLimits()
	r.Limits.MaxIterations = 1
	_, err := r.RunGoal(context.Background(), "two answers", approve, emit)
	if err == nil || !strings.Contains(err.Error(), "iteration budget") || len(g.requests) != 3 {
		t.Fatalf("budget reset between Tasks: %d %v", len(g.requests), err)
	}
}
