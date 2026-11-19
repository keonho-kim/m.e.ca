// Purpose: Verify generation recovery preserves one draft and bounds native call correction.
// Pattern: Runtime lifecycle tests with virtual time.
// Usage: go test ./internal/agent/run -run TestRecovery.
// Related: internal/agent/run/usage.go, internal/agent/run/budget.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
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

func TestRecoveryContextPreservesCheckpointObservations(t *testing.T) {
	g := &generator{replies: []string{"Inspect current state before continuing"}}
	r := fixture(g, &executor{}, &journal{})
	r.PreviousRun = &runrecord.Checkpoint{ID: "previous", Objective: "repair parser; preserve parser cancellation", Status: runrecord.Interrupted, Phase: "execution", InstructionsDigest: strings.Repeat("0", 64), Observation: "parser-check failed"}
	if _, err := r.Run(context.Background(), "continue", approve, emit); err != nil {
		t.Fatal(err)
	}
	last := fmt.Sprint(g.requests[0].Messages)
	if !strings.Contains(last, "repair parser") || !strings.Contains(last, "parser-check") || !strings.Contains(last, "preserve parser cancellation") {
		t.Fatalf("unsafe recovery context: %s", last)
	}
}

func TestRecoveryBeforeOutputAccountsEveryAttempt(t *testing.T) {
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

func TestRecoveryMalformedToolReplyRecordsFailure(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	calls := 0
	r.Generator = statisticsModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		if calls == 1 {
			return &modelapi.ToolCallError{Detail: "invalid generated arguments"}
		}
		return emit(modelapi.Event{Delta: "Need corrected arguments.", Done: true})
	})
	result, err := r.Run(context.Background(), "Create a", approve, emit)
	if err != nil || !result.HadToolFailure || result.Text != "Need corrected arguments." || calls != 2 {
		t.Fatalf("result=%+v calls=%d err=%v", result, calls, err)
	}
}

func TestRecoveryResumesDisplayedResponse(t *testing.T) {
	const draft = "Partial answer."
	const continued = draft + " More."
	cause := errors.New("all remaining attempts disconnected")
	for _, tc := range []struct {
		name, retryText, result string
		err                     error
		streams                 []string
	}{
		{"matching prefix", continued, continued, nil, []string{draft, continued}},
		{"different response", "Another answer.", "Another answer.", nil, []string{draft}},
		{"exhausted after divergence", "Another unfinished answer.", draft, cause, []string{draft}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			j := &journal{}
			r := fixture(&generator{}, &executor{}, j)
			r.Generator = attemptScript{events: []modelapi.Event{
				{Thinking: "Observed thought.", Delta: draft, Streaming: true},
				{Usage: &modelapi.TokenUsage{Input: 10, Output: 2}},
				{Retry: &modelapi.Retry{Attempt: 2, MaxAttempts: 4, Reason: "stream disconnected"}},
				{Thinking: "Observed ", Streaming: true},
				{Thinking: "thought.", Delta: tc.retryText, Done: tc.err == nil, Streaming: true},
			}, err: tc.err}
			var streams []string
			starts, notices := 0, 0
			result, err := r.Run(t.Context(), "hello", approve, func(event runrecord.Event) error {
				switch event.Kind {
				case "generation_started":
					starts++
				case "retry":
					notices++
				case "stream":
					if event.Text != "" || len(streams) != 0 {
						streams = append(streams, event.Text)
					}
				}
				return nil
			})
			if !errors.Is(err, tc.err) || recordedModelCalls(j) != 2 || starts != 1 || notices != 1 || !slices.Equal(streams, tc.streams) {
				t.Fatalf("status=%s calls=%d starts=%d notices=%d streams=%q err=%v", result.Status, recordedModelCalls(j), starts, notices, streams, err)
			}
			if len(j.messages) != 2 || j.messages[1].Incomplete != (tc.err != nil) || j.messages[1].Content != tc.result {
				t.Fatalf("unexpected saved response: %+v", j.messages)
			}
			thoughts := 0
			for _, event := range j.events {
				if event.Kind == "thinking" && event.Thinking == "Observed thought." {
					thoughts++
				}
			}
			if thoughts != 1 {
				t.Fatalf("retained Thinking entries=%d", thoughts)
			}
		})
	}
}

func TestRecoveryCorrectsToolCallsAcrossDecisions(t *testing.T) {
	j, tools := &journal{}, &executor{}
	r := fixture(&generator{}, tools, j)
	const text = "Checking the next file."
	cause := &modelapi.ToolCallError{Detail: "invalid generated arguments"}
	requests, notices := 0, 0
	r.Generator = statisticsModel(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		requests++
		if requests == 5 {
			return emit(modelapi.Event{Delta: "Both files checked.", Done: true})
		}
		if requests%2 == 1 {
			if err := emit(modelapi.Event{Delta: text, Thinking: "Check current state.", Streaming: true}); err != nil {
				return err
			}
			return cause
		}
		last := request.Messages[len(request.Messages)-1]
		if !strings.Contains(last.Content, cause.Error()) || strings.Contains(fmt.Sprint(request.Messages), "Unfinished assistant draft") {
			t.Fatalf("correction context lost its validation error or contains a replay draft: %+v", last)
		}
		call := toolapi.Call{ID: fmt.Sprintf("call_%d", requests), Name: "inspect_fixture", Arguments: []byte(fmt.Sprintf("{\"path\":\"file%d\"}", requests))}
		return emit(modelapi.Event{Done: true, Calls: []toolapi.Call{call}})
	})
	result, err := r.Run(t.Context(), "Inspect both files.", approve, func(event runrecord.Event) error {
		if event.Kind == "retry" {
			notices++
		}
		return nil
	})
	if err != nil || result.Status != runrecord.Answered || !result.HadToolFailure || requests != 5 || notices != 2 || tools.prepared != 2 || tools.executed != 2 {
		t.Fatalf("status=%s calls=%d notices=%d prepared=%d executed=%d err=%v", result.Status, requests, notices, tools.prepared, tools.executed, err)
	}
	for _, message := range j.messages {
		if message.Incomplete || strings.Contains(message.Content, text) {
			t.Fatal("a failed attempt was archived as a response")
		}
	}
}

func TestRecoveryStopsAfterOneMalformedToolCorrection(t *testing.T) {
	j, tools := &journal{}, &executor{}
	r := fixture(&generator{}, tools, j)
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 4
	const draft = "Preparing the tool call."
	cause := &modelapi.ToolCallError{Detail: "invalid generated arguments"}
	var requests, notices int
	r.Generator = statisticsModel(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		requests++
		if requests == 2 && !strings.Contains(fmt.Sprint(request.Messages), cause.Error()) {
			t.Error("correction request omitted validation feedback")
		}
		if err := emit(modelapi.Event{Delta: draft, Streaming: true}); err != nil {
			return err
		}
		return cause
	})
	result, err := r.Run(t.Context(), "hello", approve, func(event runrecord.Event) error {
		if event.Kind == "retry" {
			notices++
		}
		return nil
	})
	if !errors.Is(err, cause) || result.Status != runrecord.Stopped || !result.HadToolFailure || requests != 2 || notices != 1 || tools.prepared != 0 {
		t.Fatalf("status=%s calls=%d notices=%d tool preparations=%d err=%v", result.Status, requests, notices, tools.prepared, err)
	}
	saved := 0
	for _, message := range j.messages {
		if message.Incomplete && message.Content == draft {
			saved++
		}
	}
	if saved != 1 {
		t.Fatalf("saved %d interrupted drafts, want one", saved)
	}
}
