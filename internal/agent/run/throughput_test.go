// Purpose: Verify throughput uses real output counts and resets at each retry without storage changes.
// Pattern: Generation lifecycle tests with virtual time.
// Usage: go test ./internal/agent/run -run TestThroughput.
// Related: internal/agent/run/throughput.go, internal/agent/run/usage.go.
package run

import (
	"context"
	"encoding/json"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
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
			_, err := r.Run(context.Background(), "hello", false, approve, func(e Event) error {
				if e.Kind == "generation_started" {
					starts++
				}
				if e.TokensPerSecond != nil {
					if e.Kind == "usage_progress" {
						progress = append(progress, *e.TokensPerSecond)
					}
					if e.Kind == "usage" {
						final = append(final, *e.TokensPerSecond)
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
				if event.TokensPerSecond != nil {
					t.Fatal("transient throughput stored")
				}
				body, err := json.Marshal(event)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(body), "TokensPerSecond") {
					t.Fatal("throughput serialized")
				}
			}
		})
	}
}

func TestThroughputUnmeasurableInterval(t *testing.T) {
	if outputRate(50, 0) != nil {
		t.Fatal("zero duration produced throughput")
	}
	if rate := outputRate(0, time.Second); rate == nil || *rate != 0 {
		t.Fatal("reported zero lost")
	}
}
