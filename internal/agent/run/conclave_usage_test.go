// Purpose: Verify concurrent review retries share the call budget and only received usage reaches totals.
// Pattern: Parallel accounting integration tests.
// Usage: go test -race ./internal/agent/run -run TestConclaveRetry.
// Related: internal/agent/run/conclave_usage.go, internal/agent/run/conclave_test.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"sync/atomic"
	"testing"
)

func TestConclaveRetryBudgetAndMeasuredUsage(t *testing.T) {
	g, e, j := &generator{replies: []string{`{"tool":"write_file"}`, "a", "new"}}, &executor{}, &journal{}
	var completed atomic.Int32
	g.review = func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		if err := emit(modelapi.Event{Delta: "0", Usage: &modelapi.TokenUsage{Input: 10, Output: 1}}); err != nil {
			return err
		}
		if err := emit(modelapi.Event{Retry: &modelapi.Retry{Attempt: 2, MaxAttempts: 3, Reason: "lost connection"}}); err != nil {
			return err
		}
		completed.Add(1)
		return emit(modelapi.Event{Delta: "1", Done: true, Usage: &modelapi.TokenUsage{Input: 20, Output: 2}})
	}
	r := fixture(g, e, j)
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 8
	r.Usage = &Usage{}
	total, current := modelapi.TokenUsage{}, modelapi.TokenUsage{}
	_, err := r.Run(context.Background(), "fix", false, approve, func(event Event) error {
		if event.Kind == "usage_progress" {
			current = *event.Usage
		}
		if event.Kind == "usage" {
			total.Input += event.Usage.Input
			total.Output += event.Usage.Output
			current = modelapi.TokenUsage{}
		}
		return nil
	})
	// Decision + two fields + grounding + three ballots consume seven calls.
	// Only one of the three simultaneous retries fits the remaining call budget.
	if err == nil || !strings.Contains(err.Error(), "budget") || completed.Load() != 1 || r.Usage.ModelCalls != 8 || e.executed != 0 {
		t.Fatalf("retries=%d calls=%d exec=%d err=%v", completed.Load(), r.Usage.ModelCalls, e.executed, err)
	}
	if total.Input != 50 || total.Output != 5 || current != (modelapi.TokenUsage{}) {
		t.Fatalf("usage total=%+v current=%+v", total, current)
	}
	var persisted modelapi.TokenUsage
	for _, event := range j.events {
		if event.Kind == "usage" {
			persisted.Input += event.Usage.Input
			persisted.Output += event.Usage.Output
		}
	}
	if persisted != total {
		t.Fatalf("stored=%+v displayed=%+v", persisted, total)
	}
}
