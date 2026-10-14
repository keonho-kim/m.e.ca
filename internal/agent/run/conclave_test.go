// Purpose: Verify rejection feedback, whole-task budgets and approval ordering through real PFC phases.
// Pattern: Runtime integration tests with controlled model ballots.
// Usage: go test -race ./internal/agent/run -run TestConclave.
// Related: internal/agent/run/conclave.go, internal/agent/run/execute.go, internal/agent/run/run_test.go.
package run

import (
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"testing"
)

func TestConclaveRegeneratesWithoutSeparateRejectionLimit(t *testing.T) {
	g, e, j := &generator{purposes: []string{"check"}}, &executor{approval: true}, &journal{}
	for i := 0; i < 5; i++ {
		g.replies = append(g.replies, `{"tool":"write_file"}`, "file.txt", fmt.Sprintf("draft-%d", i))
	}
	g.replies = append(g.replies, `{"tool":"execute_command"}`, "test-command", "DONE\nfinished")
	g.review = func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		reply := "0\nmissing edge-case coverage"
		if strings.Contains(r.Messages[1].Content, "draft-4") {
			reply = "1"
		}
		return emit(modelapi.Event{Delta: reply, Done: true, Usage: &modelapi.TokenUsage{Input: 10, Output: 2}})
	}
	r := fixture(g, e, j)
	r.Settings.Scope = policy.System
	approvals := 0
	result, err := r.Run(context.Background(), "fix", false, func(_ context.Context, p toolapi.Approval) (bool, error) {
		approvals++
		if p.Candidate != nil && !p.ConclavePassed {
			t.Error("approval requested before review pass")
		}
		return true, nil
	}, emit)
	if err != nil || result.Status != Verified || e.executed != 2 || approvals != 2 || len(g.reviewRequests) != 15 {
		t.Fatalf("result=%+v err=%v executed=%d approvals=%d reviewers=%d", result, err, e.executed, approvals, len(g.reviewRequests))
	}
	if !strings.Contains(g.requests[3].Messages[len(g.requests[3].Messages)-1].Content, "missing edge-case coverage") {
		t.Fatal("next decision lost refusal reason")
	}
	reviews, proposed, started := 0, 0, 0
	total := modelapi.TokenUsage{}
	for _, event := range j.events {
		if event.Kind == "usage" {
			total.Input += event.Usage.Input
			total.Output += event.Usage.Output
		}
		if event.Monitor != nil {
			switch event.Monitor.Kind {
			case "review_rejected":
				reviews++
			case "tool_proposed":
				proposed++
			case "tool_started":
				started++
			}
		}
	}
	if reviews != 4 || proposed != 6 || started != 2 || total.Input != 150 || total.Output != 30 {
		t.Fatalf("monitor=%d/%d/%d tokens=%+v", reviews, proposed, started, total)
	}
}
func TestConclaveErrorsBlockExecutionAndCompletion(t *testing.T) {
	g, e, j := &generator{replies: []string{`{"tool":"write_file"}`, "file.txt", "new", "DONE\nunsafe completion"}}, &executor{approval: true}, &journal{}
	g.review = func(_ context.Context, _ modelapi.Request, _ func(modelapi.Event) error) error {
		return fmt.Errorf("review timeout")
	}
	r := fixture(g, e, j)
	r.Limits = DefaultLimits()
	r.Limits.MaxIterations = 2
	approvals := 0
	result, err := r.Run(context.Background(), "fix", false, func(context.Context, toolapi.Approval) (bool, error) { approvals++; return true, nil }, emit)
	if err == nil || result.Status != Stopped || e.executed != 0 || approvals != 0 {
		t.Fatalf("result=%+v err=%v exec=%d approval=%d", result, err, e.executed, approvals)
	}
	var cp *Checkpoint
	for _, event := range j.events {
		if event.Checkpoint != nil {
			cp = event.Checkpoint
		}
	}
	if cp == nil || !cp.PendingReview || cp.Conclave == nil || cp.Conclave.Complete {
		t.Fatalf("missing blocked review: %+v", cp)
	}
}
func TestConclaveJournalFailureCannotProceed(t *testing.T) {
	g, e, j := &generator{replies: []string{`{"tool":"write_file"}`, "file.txt", "new"}}, &executor{}, &journal{failKind: "conclave"}
	_, err := fixture(g, e, j).Run(context.Background(), "fix", false, approve, emit)
	if err == nil || e.executed != 0 {
		t.Fatalf("execution followed missing audit: %d %v", e.executed, err)
	}
}
