// Purpose: Suppress duplicate telemetry while retaining cost-only changes and final measurements.
// Pattern: Runtime stream accounting behavior test.
// Usage: Agent tests supply actual provider snapshots without network requests.
// Related: internal/agent/run/usage.go, internal/agent/model/usage.go.
package run

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestUsageProgressKeepsCostChangesAndOneFinal(t *testing.T) {
	j := &journal{}
	r := fixture(&generator{}, &executor{}, j)
	zero, charge := 0.0, 0.001
	r.Generator = attemptScript{events: []modelapi.Event{
		{Usage: &modelapi.TokenUsage{Input: 10, Output: 2}},
		{Usage: &modelapi.TokenUsage{Input: 10, Output: 2}},
		{Usage: &modelapi.TokenUsage{Input: 10, Output: 2, Cost: &zero}},
		{Usage: &modelapi.TokenUsage{Input: 10, Output: 2, Cost: &charge}},
		{Done: true, Usage: &modelapi.TokenUsage{Input: 10, Output: 2, Cost: &charge}},
	}}
	var progress []modelapi.TokenUsage
	finals, consumed := 0, 0
	s := execution{runner: r, limits: DefaultLimits(), emit: func(e runrecord.Event) error {
		if e.Kind == "usage_progress" {
			progress = append(progress, *e.Usage)
		}
		if e.Kind == "usage" {
			finals++
		}
		return nil
	}}
	if err := s.generate(context.Background(), modelapi.Request{}, inference.Agent, func(modelapi.Event) error { consumed++; return nil }); err != nil {
		t.Fatal(err)
	}
	if len(progress) != 3 || finals != 1 || consumed != 5 || progress[0].Cost != nil || progress[1].Cost == nil || *progress[1].Cost != 0 || *progress[2].Cost != charge {
		t.Fatal("actual measurements or completion changed", progress, finals, consumed)
	}
}
