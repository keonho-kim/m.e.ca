// Purpose: Verify request summaries retain measured attempts, weighted generation time and independent request totals.
// Pattern: Runtime contract tests with virtual time.
// Usage: go test ./internal/agent/run -run TestRunStatistics.
// Related: internal/agent/run/statistics.go, internal/agent/run/usage.go, internal/agent/run/journal.go.
package run

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type statisticsModel func(context.Context, modelapi.Request, func(modelapi.Event) error) error

func (g statisticsModel) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return g(ctx, request, emit)
}

func TestRunStatisticsAttemptsAndRequests(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		j := &journal{}
		r := fixture(&generator{}, &executor{}, j)
		r.Generator = statisticsModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
			for _, output := range []int64{2, 20} {
				time.Sleep(time.Second)
				if err := emit(modelapi.Event{Usage: &modelapi.TokenUsage{Input: 100, Output: output}}); err != nil {
					return err
				}
			}
			if err := emit(modelapi.Event{Retry: &modelapi.Retry{Attempt: 2, MaxAttempts: 3}}); err != nil {
				return err
			}
			time.Sleep(4 * time.Second)
			if err := emit(modelapi.Event{Delta: "answer", Done: true, Usage: &modelapi.TokenUsage{Input: 200, Output: 20}}); err != nil {
				return err
			}
			time.Sleep(10 * time.Second)
			return nil
		})
		var summaries []runrecord.Event
		for i := 0; i < 2; i++ {
			if i == 1 {
				r.Generator = throughputGenerator{}
			}
			_, err := r.Run(context.Background(), "hello", approve, func(event runrecord.Event) error {
				if event.Statistics != nil {
					summaries = append(summaries, event)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		if len(summaries) != 2 {
			t.Fatalf("summaries: %+v", summaries)
		}
		for i, want := range []struct{ elapsed, generation, input, output int64 }{{16000, 6000, 300, 40}, {14000, 4000, 100, 60}} {
			event := summaries[i]
			if err := event.Check(); err != nil {
				t.Fatal(err)
			}
			got := event.Statistics
			if got.ElapsedMillis != want.elapsed || got.GenerationMillis != want.generation || got.Usage == nil || got.Usage.Input != want.input || got.Usage.Output != want.output {
				t.Fatalf("request %d: %+v usage=%+v", i, got, got.Usage)
			}
		}
		stored := 0
		for _, event := range j.events {
			if event.Statistics != nil {
				stored++
			}
		}
		if stored != 2 || summaries[0].RunID == summaries[1].RunID {
			t.Fatal("request summaries were duplicated or reused")
		}
	})
}

func TestRunStatisticsMissingZeroAndInterruptedUsage(t *testing.T) {
	for _, test := range []struct {
		name  string
		usage *modelapi.TokenUsage
		err   error
	}{{"missing", nil, nil}, {"reported zero", &modelapi.TokenUsage{}, nil}, {"interrupted", &modelapi.TokenUsage{Input: 20, Output: 4}, context.Canceled}} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := fixture(&generator{}, &executor{}, &journal{})
				r.Generator = statisticsModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
					time.Sleep(time.Second)
					if err := emit(modelapi.Event{Delta: "answer", Done: test.err == nil, Usage: test.usage}); err != nil {
						return err
					}
					return test.err
				})
				var summary *runrecord.Statistics
				_, err := r.Run(context.Background(), "hello", approve, func(event runrecord.Event) error {
					if event.Statistics != nil {
						summary = event.Statistics
					}
					return nil
				})
				if !errors.Is(err, test.err) || summary == nil || summary.ElapsedMillis != 1000 {
					t.Fatalf("summary=%+v err=%v", summary, err)
				}
				if test.usage == nil {
					if summary.Usage != nil || summary.GenerationMillis != 0 {
						t.Fatal("unreported usage was estimated")
					}
				} else if summary.Usage == nil || *summary.Usage != *test.usage || summary.GenerationMillis != 1000 {
					t.Fatalf("measured usage lost: %+v", summary)
				}
			})
		})
	}
}

func TestRunStatisticsIncludesCompaction(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := fixture(&generator{}, &executor{}, &journal{})
		r.Generator = throughputGenerator{}
		s := execution{runner: r, limits: DefaultLimits(), usage: &requestBudget{}, emit: emit, statistics: runStatistics{started: time.Now()}}
		s.usage.begin(s.limits)
		for _, account := range []bool{true, false} {
			if err := s.generate(context.Background(), modelapi.Request{}, account, func(modelapi.Event) error { return nil }); err != nil {
				t.Fatal(err)
			}
		}
		got := s.statistics.snapshot(time.Now())
		if got.ElapsedMillis != 28000 || got.GenerationMillis != 8000 || got.Usage == nil || *got.Usage != (modelapi.TokenUsage{Input: 200, Output: 120}) {
			t.Fatalf("compaction usage omitted: %+v", got)
		}
	})
}
