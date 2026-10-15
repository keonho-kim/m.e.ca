// Purpose: Verify rejection feedback, whole-execution budgets and approval ordering through real native call phases.
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
	g, e, j := &generator{}, &executor{approval: true}, &journal{}
	for i := 0; i < 5; i++ {
		g.replies = append(g.replies, nativeCall("write_file", map[string]any{"path": "file.txt", "content": fmt.Sprintf("draft-%d", i)}))
	}
	g.replies = append(g.replies, nativeCall("execute_command", map[string]any{"command": "test-command"}), "finished")
	g.review = func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		reply := `{"decision":"reject","findings":[{"path":"file.txt","side":"request","line":0,"reason":"missing edge-case coverage"}]}`
		if strings.Contains(r.Messages[1].Content, "draft-4") {
			reply = `{"decision":"pass"}`
		}
		return emit(modelapi.Event{Delta: reply, Done: true, Usage: &modelapi.TokenUsage{Input: 10, Output: 2}})
	}
	r := fixture(g, e, j)
	r.Settings.Scope = policy.System
	approvals := 0
	result, err := r.Run(context.Background(), "fix", func(_ context.Context, p toolapi.Approval) (bool, error) {
		approvals++
		if p.Candidate != nil && !p.ConclavePassed {
			t.Error("approval requested before review pass")
		}
		return true, nil
	}, emit)
	if err != nil || result.Status != Answered || e.executed != 2 || approvals != 2 || len(g.reviewRequests) != 5 {
		t.Fatalf("result=%+v err=%v executed=%d approvals=%d reviewers=%d", result, err, e.executed, approvals, len(g.reviewRequests))
	}
	var feedback strings.Builder
	for _, m := range g.requests[3].Messages {
		feedback.WriteString(m.Content)
	}
	if !strings.Contains(feedback.String(), "missing edge-case coverage") {
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
	if reviews != 4 || proposed != 6 || started != 2 || total.Input != 50 || total.Output != 10 {
		t.Fatalf("monitor=%d/%d/%d tokens=%+v", reviews, proposed, started, total)
	}
}
func TestConclaveErrorsBlockExecutionAndCompletion(t *testing.T) {
	g, e, j := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file.txt", "content": "new"}), "unsafe completion"}}, &executor{approval: true}, &journal{}
	g.review = func(_ context.Context, _ modelapi.Request, _ func(modelapi.Event) error) error {
		return fmt.Errorf("review timeout")
	}
	r := fixture(g, e, j)
	r.Limits = DefaultLimits()
	r.Limits.MaxIterations = 2
	approvals := 0
	result, err := r.Run(context.Background(), "fix", func(context.Context, toolapi.Approval) (bool, error) { approvals++; return true, nil }, emit)
	if err != nil || result.Status != Stopped || e.executed != 0 || approvals != 0 {
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
	g, e, j := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "file.txt", "content": "new"})}}, &executor{}, &journal{failKind: "conclave"}
	_, err := fixture(g, e, j).Run(context.Background(), "fix", approve, emit)
	if err == nil || e.executed != 0 {
		t.Fatalf("execution followed missing audit: %d %v", e.executed, err)
	}
}

func TestConclaveReusesUnchangedRejectedCandidate(t *testing.T) {
	call := nativeCall("write_file", map[string]any{"path": "file.txt", "content": "bad"})
	g, e, j := &generator{replies: []string{call, call, "Cannot resolve the finding", "Blocked"}}, &executor{}, &journal{}
	g.review = func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		return emit(modelapi.Event{Delta: `{"decision":"reject","findings":[{"path":"file.txt","side":"request","line":0,"reason":"Incorrect requested content"}]}`, Done: true})
	}
	result, err := fixture(g, e, j).Run(context.Background(), "fix", approve, emit)
	if err != nil || result.Status != Stopped || len(g.reviewRequests) != 1 || e.executed != 0 {
		t.Fatalf("%+v reviews=%d executed=%d %v", result, len(g.reviewRequests), e.executed, err)
	}
	cached := false
	for _, event := range j.events {
		if event.Conclave != nil && event.Conclave.Cached {
			cached = true
		}
	}
	if !cached {
		t.Fatal("reused review was not audited")
	}
}
