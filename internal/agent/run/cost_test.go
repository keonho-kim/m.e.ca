// Purpose: Verify retries, interrupted calls and compaction preserve each reported charge once.
// Pattern: Runtime accounting contract tests.
// Usage: go test ./internal/agent/run -run TestCost.
// Related: internal/agent/run/usage.go, internal/agent/run/statistics.go.
package run

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/inference"
	"math"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestCostRetriesAndInterruption(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		t.Run(map[bool]string{false: "retry", true: "interrupted"}[interrupted], func(t *testing.T) {
			j := &journal{}
			r := fixture(&generator{}, &executor{}, j)
			r.Generator = statisticsModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
				cost := 0.001
				usage := modelapi.TokenUsage{Input: 10, Output: 2, Cost: &cost}
				if err := emit(modelapi.Event{Usage: &usage}); err != nil {
					return err
				}
				cost = 0.002
				if err := emit(modelapi.Event{Usage: &usage}); err != nil {
					return err
				}
				cost = 99 // A producer cannot mutate an already accepted measurement.
				if interrupted {
					return context.Canceled
				}
				if err := emit(modelapi.Event{Retry: &modelapi.Retry{Attempt: 2, MaxAttempts: 3}}); err != nil {
					return err
				}
				return emit(modelapi.Event{Delta: "finished", Done: true, Usage: &modelapi.TokenUsage{Input: 20, Output: 3, Cost: new(0.003)}})
			})
			var summary *runrecord.Statistics
			_, err := r.Run(context.Background(), "hello", approve, func(event runrecord.Event) error {
				if event.Statistics != nil {
					summary = event.Statistics
				}
				return nil
			})
			want, count := 0.005, 2
			if interrupted {
				want, count = 0.002, 1
			}
			if (err != nil) != interrupted || interrupted && !errors.Is(err, context.Canceled) {
				t.Fatalf("unexpected termination: %v", err)
			}
			if summary == nil || summary.Usage == nil || summary.Usage.Cost == nil || math.Abs(*summary.Usage.Cost-want) > 1e-12 {
				t.Fatalf("charge lost or duplicated: %+v", summary)
			}
			persisted := 0
			for _, event := range j.events {
				if event.Kind == "usage" {
					persisted++
				}
			}
			if persisted != count {
				t.Fatalf("persisted %d attempts, want %d", persisted, count)
			}
		})
	}
}

func TestCostCompactionUsesSameAccounting(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	r.Generator = statisticsModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		return emit(modelapi.Event{Done: true, Usage: &modelapi.TokenUsage{Input: 10, Output: 2, Cost: new(0.001)}})
	})
	s := execution{runner: r, limits: DefaultLimits(), usage: &requestBudget{}, emit: emit, statistics: runStatistics{started: time.Now()}}
	s.usage.begin(s.limits)
	for _, role := range []inference.Role{inference.Agent, inference.Compaction} {
		if err := s.generate(context.Background(), modelapi.Request{}, role, func(modelapi.Event) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := s.statistics.snapshot(time.Now())
	*s.statistics.usage.Cost = 9
	if snapshot.Usage.Cost == nil || *snapshot.Usage.Cost != 0.002 {
		t.Fatal("compaction cost omitted or snapshot aliased")
	}
}
