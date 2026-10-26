// Purpose: Verify active model replies survive elapsed budgets and retries discard drafts but retain measured usage.
// Pattern: Runtime lifecycle tests with virtual time.
// Usage: go test ./internal/agent/run -run TestRecovery.
// Related: internal/agent/run/usage.go, internal/agent/run/budget.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type recoveryGenerator struct{ slow bool }

func (g recoveryGenerator) Generate(ctx context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
	if g.slow {
		if err := emit(modelapi.Event{Delta: "", Streaming: true}); err != nil {
			return err
		}
		for range 5 {
			time.Sleep(500 * time.Millisecond)
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := emit(modelapi.Event{Thinking: "working", Streaming: true}); err != nil {
				return err
			}
		}
	} else {
		for _, event := range []modelapi.Event{
			{Delta: "wrong partial", Thinking: "old thought", Streaming: true},
			{Usage: &modelapi.TokenUsage{Input: 10, Output: 2}},
			{Retry: &modelapi.Retry{Attempt: 2, MaxAttempts: 3, Reason: "stream disconnected"}},
		} {
			if err := emit(event); err != nil {
				return err
			}
		}
	}
	if err := emit(modelapi.Event{Delta: "recovered", Streaming: true}); err != nil {
		return err
	}
	return emit(modelapi.Event{Done: true, Usage: &modelapi.TokenUsage{Input: 10, Output: 3}})
}

func TestRecoveryContextPreservesCheckpointWithoutDraftReplay(t *testing.T) {
	g := &generator{replies: []string{"Inspect current state before continuing"}}
	r := fixture(g, &executor{}, &journal{})
	r.PreviousRun = &runrecord.Checkpoint{ID: "previous", Objective: "repair parser; preserve parser cancellation", Status: runrecord.Interrupted, Phase: "execution", InstructionsDigest: strings.Repeat("0", 64), Observation: "parser-check failed", Arguments: map[string]json.RawMessage{"content": json.RawMessage(`"UNEXECUTED_DRAFT"`)}}
	if _, err := r.Run(context.Background(), "continue", approve, emit); err != nil {
		t.Fatal(err)
	}
	last := fmt.Sprint(g.requests[0].Messages)
	if !strings.Contains(last, "repair parser") || !strings.Contains(last, "parser-check") || !strings.Contains(last, "preserve parser cancellation") || strings.Contains(last, "UNEXECUTED_DRAFT") {
		t.Fatalf("unsafe recovery context: %s", last)
	}
}

func TestRecoveryDiscardsPartialAndAccountsEveryAttempt(t *testing.T) {
	j := &journal{}
	r := fixture(&generator{}, &executor{}, j)
	r.Generator = recoveryGenerator{}
	result, err := r.Run(context.Background(), "hello", approve, emit)
	if err != nil || result.Text != "recovered" || recordedModelCalls(j) != 2 {
		t.Fatalf("result=%+v calls=%d err=%v", result, recordedModelCalls(j), err)
	}
	var input, output int64
	for _, event := range j.events {
		if event.Kind == "usage" {
			input += event.Usage.Input
			output += event.Usage.Output
		}
		if event.Kind == "thinking" {
			t.Fatal("failed attempt's thinking committed")
		}
	}
	if input != 20 || output != 5 {
		t.Fatalf("input=%d output=%d", input, output)
	}
}

func TestRecoveryActiveCallSurvivesElapsedTaskBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := fixture(&generator{}, &executor{}, &journal{})
		r.Generator = recoveryGenerator{slow: true}
		r.Limits = DefaultLimits()
		r.Limits.MaxSeconds, r.Limits.MaxModelSeconds = 1, 1
		result, err := r.Run(context.Background(), "hello", approve, emit)
		if err != nil || result.Text != "recovered" {
			t.Fatalf("active call interrupted: %+v %v", result, err)
		}
	})
}

func TestRecoveryRespectsCallBudget(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	r.Generator = recoveryGenerator{}
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 1
	_, err := r.Run(context.Background(), "hello", approve, emit)
	if err == nil || !strings.Contains(err.Error(), "budget exhausted before provider retry") {
		t.Fatalf("budget error=%v", err)
	}
}
