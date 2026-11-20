// Purpose: Measure first output and token throughput for one attempt and describe its input budget.
// Pattern: Attempt-local sampler with explicit observation timestamps.
// Usage: generate publishes live timing and persists its final measurements with model usage.
// Related: internal/agent/record/model_activity.go, internal/agent/run/usage.go, internal/agent/compact/usage.go.
package run

import (
	"time"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

type attemptMetrics struct {
	activity               record.ModelActivity
	baseTime               time.Time
	baseTokens, lastTokens int64
}

// observe uses the first post-output counter as the baseline. Repeated counts
// carry no new timing information. A server decode duration takes precedence.
func (m *attemptMetrics) observe(event modelapi.Event, now time.Time) bool {
	first := m.activity.FirstOutput.IsZero() && (event.Delta != "" || event.Thinking != "" || len(event.Calls) > 0 || event.Progress)
	if first {
		m.activity.FirstOutput = now
	}
	if event.Usage == nil {
		return first
	}
	output := event.Usage.Output
	if event.DecodeDuration > 0 && output > 0 {
		m.activity.Speed = record.OutputSpeed{Source: "provider", Throughput: record.Throughput{Output: output, Duration: event.DecodeDuration}}
		return first
	}
	if m.activity.FirstOutput.IsZero() || m.activity.Speed.Source == "provider" {
		return first
	}
	if m.baseTime.IsZero() || output < m.lastTokens {
		m.baseTime, m.baseTokens, m.lastTokens = now, output, output
		m.activity.Speed = record.OutputSpeed{}
	} else if output > m.lastTokens && now.After(m.baseTime) {
		m.lastTokens = output
		m.activity.Speed = record.OutputSpeed{Source: "received", Throughput: record.Throughput{Output: output - m.baseTokens, Duration: now.Sub(m.baseTime)}}
	}
	return first
}

func measuredContext(r modelapi.Request, b modelapi.InputBudget, wire int64, u *modelapi.TokenUsage) *modelapi.ContextMeasurement {
	c := &modelapi.ContextMeasurement{Window: b.Window, OutputBudget: b.OutputTokens}
	switch {
	case u != nil:
		c.Source, c.Input = "usage", u.Input
	case b.InputTokens > 0:
		c.Source, c.Input = "tokenizer", int64(b.InputTokens)
	default:
		if wire <= 0 {
			wire = modelapi.TextBytes(r)
		}
		c.Input, c.Source = compact.EstimateBytes(wire).Amount, "estimate"
	}
	return c
}
