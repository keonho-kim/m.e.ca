// Purpose: Verify attempt measurements preserve real counts and exclude completion delays without storage changes.
// Pattern: Generation lifecycle tests with virtual time.
// Usage: go test ./internal/agent/run -run TestThroughput.
// Related: internal/agent/record/throughput.go, internal/agent/run/usage.go.
package run

import (
	"context"
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type throughputGenerator struct{ retry bool }

func (g throughputGenerator) Generate(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
	time.Sleep(2 * time.Second)
	if err := emit(modelapi.Event{Usage: &modelapi.TokenUsage{Input: 100, Output: 20}}); err != nil {
		return err
	}
	if g.retry {
		time.Sleep(5 * time.Second)
		if err := emit(modelapi.Event{Retry: &modelapi.Retry{Attempt: 2, MaxAttempts: 3}}); err != nil {
			return err
		}
	}
	time.Sleep(2 * time.Second)
	count := int64(60)
	if g.retry {
		count = 10
	}
	if err := emit(modelapi.Event{Delta: "answer", Done: true, Usage: &modelapi.TokenUsage{Input: 100, Output: count}}); err != nil {
		return err
	}
	time.Sleep(10 * time.Second) // Completion handling must not dilute the last measured snapshot.
	return nil
}

func TestThroughputMeasuredPerAttempt(t *testing.T) {
	for _, retry := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			j := &journal{}
			r := fixture(&generator{}, &executor{}, j)
			r.Generator = throughputGenerator{retry: retry}
			var progress, final []float64
			starts := 0
			_, err := r.Run(context.Background(), "hello", approve, func(e runrecord.Event) error {
				if e.Kind == "generation_started" {
					starts++
				}
				if rate := runrecord.MeasuredThroughput(e).Rate(); rate != nil {
					if e.Kind == "usage_progress" {
						progress = append(progress, *rate)
					}
					if e.Kind == "usage" {
						final = append(final, *rate)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			want := 15.0
			count := 1
			if retry {
				want = 5
				count = 2
			}
			if len(progress) != 2 || progress[0] != 10 || progress[1] != want || starts != count || len(final) != count || final[len(final)-1] != want {
				t.Fatalf("starts=%d progress=%v final=%v", starts, progress, final)
			}
			for _, event := range j.events {
				if event.MeasuredDuration != 0 {
					t.Fatal("transient throughput stored")
				}
				body, err := json.Marshal(event)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(body), "MeasuredDuration") {
					t.Fatal("throughput serialized")
				}
			}
		})
	}
}

func TestThroughputDuplicateUsageKeepsItsMeasuredInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := fixture(&generator{}, &executor{}, &journal{})
		r.Generator = statisticsModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
			time.Sleep(2 * time.Second)
			if err := emit(modelapi.Event{Usage: &modelapi.TokenUsage{Input: 100, Output: 20, Cost: new(0.001)}}); err != nil {
				return err
			}
			time.Sleep(10 * time.Second)
			return emit(modelapi.Event{Delta: "answer", Done: true, Usage: &modelapi.TokenUsage{Input: 100, Output: 20, Cost: new(0.002)}})
		})
		var measured []time.Duration
		var summary *runrecord.Statistics
		_, err := r.Run(context.Background(), "hello", approve, func(event runrecord.Event) error {
			if event.Usage != nil {
				measured = append(measured, event.MeasuredDuration)
			}
			if event.Statistics != nil {
				summary = event.Statistics
			}
			return nil
		})
		if err != nil || len(measured) != 3 {
			t.Fatalf("measurements=%v err=%v", measured, err)
		}
		for _, duration := range measured {
			if duration != 2*time.Second {
				t.Fatalf("unchanged token count was diluted by completion delay: %v", measured)
			}
		}
		if summary == nil || summary.GenerationMillis != 2000 || summary.ElapsedMillis != 12000 || summary.Usage.Cost == nil || *summary.Usage.Cost != 0.002 {
			t.Fatalf("duplicate final usage lost actual totals or final cost: %+v", summary)
		}
	})
}
