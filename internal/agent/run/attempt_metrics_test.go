// Purpose: Verify first-output latency, per-attempt rate and context provenance independently of total work time.
// Pattern: Virtual-time model lifecycle tests.
// Usage: go test ./internal/agent/run -run TestAttemptMetrics.
// Related: internal/agent/run/usage.go, internal/agent/run/attempt_metrics.go.
package run

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	record "github.com/keonho-kim/meca/internal/agent/record"
)

func TestAttemptMetrics(t *testing.T) {
	for _, delay := range []time.Duration{0, 4 * time.Second} {
		synctest.Test(t, func(t *testing.T) {
			j := &journal{}
			r := fixture(&generator{}, &executor{}, j)
			r.Generator = statisticsModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
				if err := emit(modelapi.Event{InputBudget: &modelapi.InputBudget{Window: 4096, InputTokens: 100, OutputTokens: 900}}); err != nil {
					return err
				}
				time.Sleep(delay)
				if err := emit(modelapi.Event{Delta: "a", Streaming: true, Usage: &modelapi.TokenUsage{Input: 100, Output: 10}}); err != nil {
					return err
				}
				time.Sleep(time.Second)
				return emit(modelapi.Event{Delta: "b", Done: true, Usage: &modelapi.TokenUsage{Input: 100, Output: 110}})
			})
			var end record.Event
			_, err := r.Run(t.Context(), "hello", approve, func(e record.Event) error {
				if e.Kind == "usage" {
					end = e
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if end.Activity == nil || end.Activity.Role != inference.Agent || end.Activity.Speed.Source != "received" || *end.Activity.Speed.Throughput.Rate() != 100 {
				t.Fatalf("speed=%+v", end.Activity)
			}
			g := end.Generation
			if g.FirstOutputMillis == nil || *g.FirstOutputMillis != delay.Milliseconds() || end.MeasuredDuration != delay+time.Second || g.Context.Input != 100 || g.Context.Source != "usage" || g.Context.Window != 4096 || g.Context.OutputBudget != 900 {
				t.Fatalf("generation=%+v duration=%s", g, end.MeasuredDuration)
			}
		})
	}
}
