// Purpose: Account native LAT phase usage once without charging the parent context.
// Pattern: Runtime accounting boundary test.
// Usage: go test ./internal/agent/run -run TestLATUsage.
// Related: internal/agent/run/usage.go, internal/agent/run/lat_fixture_test.go.
package run

import (
	"context"
	"math"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestLATUsageSharesCostButNotMainContext(t *testing.T) {
	store := &usageRecords{}
	r, _, _, _ := latFixture("A plain answer")
	r.Journal = &latJournal{&store.journal}
	r.ReviewMemory = &journalMemory{journal: &store.journal}
	r.Memory = &compact.Controller{Repository: store, Session: "s", Prompt: prompt.Defaults().Compaction, Settings: compact.Defaults(), Redact: r.Redact}
	result, err := r.Run(context.Background(), "Explain", approve, emit)
	if err != nil || result.Status != runrecord.Answered {
		t.Fatalf("LAT run: %+v %v", result, err)
	}
	if len(store.counts) != 1 || store.counts[0].Amount != 105 {
		t.Fatalf("private phase usage polluted parent compaction: %+v", store.counts)
	}
	roles := map[inference.Role]int{}
	var parent *runrecord.Statistics
	usageEvents := 0
	for _, event := range store.events {
		if err := event.Check(); err != nil {
			t.Fatal("durable generation rejected", err)
		}
		if event.Kind == "usage" {
			usageEvents++
			roles[event.Generation.Role]++
		}
		if !event.Internal && event.Checkpoint != nil && event.Checkpoint.Phase == "finished" {
			parent = event.Statistics
		}
	}
	if usageEvents != 5 || roles[inference.Agent] != 3 || roles[inference.LAT] != 2 {
		t.Fatalf("phase usage omitted, duplicated or assigned to the wrong role: events=%d roles=%v", usageEvents, roles)
	}
	if parent == nil || parent.Usage == nil || parent.Usage.Input != 500 || parent.Usage.Output != 25 || parent.Usage.Cost == nil || math.Abs(*parent.Usage.Cost-0.005) > 1e-12 {
		t.Fatalf("parent statistics did not aggregate each phase once: %+v", parent)
	}
}

func TestLATRetryAccountsEachAttemptOnce(t *testing.T) {
	r, model, _, store := latFixture("Completed answer")
	model.handlers["plan"] = func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		for _, event := range []modelapi.Event{
			{Usage: &modelapi.TokenUsage{Input: 10, Output: 2}},
			{Retry: &modelapi.Retry{Attempt: 2, MaxAttempts: 2}},
			{Delta: "INITIAL_PLAN", Done: true, Usage: &modelapi.TokenUsage{Input: 20, Output: 3}},
		} {
			if err := send(event); err != nil {
				return err
			}
		}
		return nil
	}
	result, err := r.Run(context.Background(), "Explain", approve, emit)
	if err != nil || result.Text != "Completed answer" || recordedModelCalls(store) != 6 {
		t.Fatalf("retry budget: %+v calls=%d err=%v", result, recordedModelCalls(store), err)
	}
	usageEvents := 0
	var statistics *runrecord.Statistics
	for _, event := range store.events {
		if event.Kind == "usage" {
			usageEvents++
		}
		if !event.Internal && event.Checkpoint != nil && event.Checkpoint.Phase == "finished" {
			statistics = event.Statistics
		}
	}
	if usageEvents != 6 || statistics == nil || statistics.Usage == nil || statistics.Usage.Input != 430 || statistics.Usage.Output != 25 {
		t.Fatalf("retry usage omitted or duplicated: events=%d statistics=%+v", usageEvents, statistics)
	}
}

func TestLATBudgetIsSharedAndReviewRemainsAdvisory(t *testing.T) {
	r, model, _, store := latFixture("Completed answer")
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 4
	result, err := r.Run(context.Background(), "Explain", approve, emit)
	if err != nil || result.Text != "Completed answer" || len(model.requests["review"]) != 0 || recordedModelCalls(store) != 4 {
		t.Fatalf("shared budget: result=%+v phases=%v err=%v", result, model.scopes, err)
	}
	found := false
	for _, event := range store.events {
		found = found || event.Kind == "lat_unavailable"
	}
	if !found {
		t.Fatal("budget prevented review without reporting it")
	}
}
