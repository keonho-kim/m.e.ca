// Purpose: Verify parallel ballots use the same model and fail closed on reviewer errors.
// Pattern: Concurrent use-case contract tests.
// Usage: go test -race ./internal/agent/conclave.
// Related: internal/agent/conclave/review.go, internal/agent/conclave/vote.go.
package conclave

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func reviewInput(t *testing.T) Input {
	t.Helper()
	call := tool.Call{ID: "c", Name: "write_file", Arguments: []byte(`{"path":"a","content":"new"}`)}
	record, err := grounding.Parse("- Evidence: Read a.\n- Action: Replace old content.\n- Verify: Run tests.", call)
	if err != nil {
		t.Fatal(err)
	}
	return Input{GoalObjective: "fix a", TaskObjective: "update a", Monitor: "read completed", Grounding: record, Call: call, Candidate: tool.BindCandidate(call, []tool.FileDelta{{Path: "a", Added: 40, Deleted: 40, Diff: "-old\n+new"}})}
}
func TestParallelReviewQuorumAndFailure(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(fmt.Sprint(broken), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var entered atomic.Int32
			ready := make(chan struct{})
			on := true
			result, err := Review(ctx, reviewInput(t), modelapi.Request{ModelID: "serving-alias", Thinking: &on, ReasoningEffort: "high"}, 100000, func(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
				if r.ModelID != "serving-alias" || r.Thinking == nil || !*r.Thinking || r.ReasoningEffort != "high" {
					t.Errorf("model controls lost: %+v", r)
				}
				index := entered.Add(1)
				if index == 6 {
					close(ready)
				}
				select {
				case <-ready:
				case <-ctx.Done():
					return ctx.Err()
				}
				if broken && index == 6 {
					return fmt.Errorf("idle timeout")
				}
				reply := "1"
				if index > 4 {
					reply = "0\nneeds a focused test"
				}
				return emit(modelapi.Event{Delta: reply, Done: true})
			}, func(s string) string { return s })
			if err != nil || result.Reviews != 6 || result.Approvals != 4 || result.Required != 4 || result.Passed == broken || result.Complete == broken {
				t.Fatalf("ballot=%+v err=%v", result, err)
			}
			if !result.Passed && !strings.Contains(result.Feedback(), "idle timeout") {
				t.Fatal("failure reason missing")
			}
		})
	}
}
func TestReviewCancellationJoinsWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var entered, exited atomic.Int32
	result, err := Review(ctx, reviewInput(t), modelapi.Request{ModelID: "m"}, 100000, func(ctx context.Context, _ modelapi.Request, _ func(modelapi.Event) error) error {
		defer exited.Add(1)
		if entered.Add(1) == 6 {
			cancel()
		}
		<-ctx.Done()
		return ctx.Err()
	}, func(s string) string { return s })
	if err == nil || result.Passed || exited.Load() != 6 {
		t.Fatalf("not joined: %d %+v %v", exited.Load(), result, err)
	}
}
