// Purpose: Keep measured review costs in session totals and outside main-context compaction counters.
// Pattern: Runtime accounting boundary tests.
// Usage: go test ./internal/agent/run -run TestLATUsage.
// Related: internal/agent/run/usage.go, internal/agent/run/request_usage_test.go.
package run

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestLATUsageSharesCostButNotMainContext(t *testing.T) {
	store := &usageRecords{}
	r := fixture(&generator{}, &executor{}, &store.journal)
	cost := 0.001
	generator := attemptScript{events: []modelapi.Event{{Delta: "NO_FINDINGS", Done: true, Usage: &modelapi.TokenUsage{Input: 100, Output: 5, Cost: &cost}}}}
	r.Generator = generator
	r.Memory = &compact.Controller{Repository: store, Session: "s"}
	s := execution{runner: r, reviewGenerator: generator, limits: DefaultLimits(), usage: &requestBudget{}, emit: emit, statistics: runStatistics{started: time.Now()}}
	s.usage.begin(s.limits)
	for _, role := range []inference.Role{inference.Agent, inference.Compaction, inference.LAT} {
		if err := s.generate(context.Background(), modelapi.Request{}, role, func(modelapi.Event) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if len(store.counts) != 1 || store.counts[0].Amount != 105 {
		t.Fatalf("review polluted main compaction usage: %+v", store.counts)
	}
	usage := s.statistics.snapshot(time.Now()).Usage
	if usage == nil || usage.Input != 300 || usage.Output != 15 || usage.Cost == nil || math.Abs(*usage.Cost-0.003) > 1e-12 {
		t.Fatalf("review usage omitted or duplicated: %+v", usage)
	}
	roles := map[inference.Role]int{}
	for _, event := range store.events {
		if err := event.Check(); err != nil {
			t.Fatal("durable generation rejected", err)
		}
		if event.Generation != nil {
			roles[event.Generation.Role]++
		}
	}
	if roles[inference.LAT] != 1 || roles[inference.Agent] != 1 || roles[inference.Compaction] != 1 {
		t.Fatal("generation role provenance lost", roles)
	}
}

func TestLATBudgetDoesNotTurnCompletedWorkIntoAFailure(t *testing.T) {
	r, model, _, store := latFixture(nativeCall("inspect_fixture", map[string]any{"path": "a"}), "Observed a")
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 2
	result, err := r.Run(context.Background(), "Inspect a", approve, emit)
	if err != nil || result.Text != "Observed a" || len(model.reviews) != 0 || recordedModelCalls(store) != 2 {
		t.Fatalf("optional review exceeded shared budget: %+v %v", result, err)
	}
	found := false
	for _, event := range store.events {
		found = found || event.Kind == "lat_unavailable"
	}
	if !found {
		t.Fatal("budget prevented review without reporting it")
	}
}
