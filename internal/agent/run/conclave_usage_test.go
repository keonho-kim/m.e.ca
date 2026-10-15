// Purpose: Verify review retries share the call budget and only received usage reaches totals.
// Pattern: Shared budget accounting integration tests.
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
	g, e, j := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"})}}, &executor{}, &journal{}
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
	r.Limits.MaxModelCalls = 2
	r.Usage = &Usage{}
	total, current := modelapi.TokenUsage{}, modelapi.TokenUsage{}
	result, err := r.Run(context.Background(), "fix", approve, func(event Event) error {
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
	// The main call and review consume the shared budget before retry.
	if err != nil || result.Status != Stopped || !strings.Contains(result.Text, "budget") || completed.Load() != 0 || r.Usage.ModelCalls != 2 || e.executed != 0 {
		t.Fatalf("retries=%d calls=%d exec=%d err=%v", completed.Load(), r.Usage.ModelCalls, e.executed, err)
	}
	if total.Input != 10 || total.Output != 1 || current != (modelapi.TokenUsage{}) {
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
