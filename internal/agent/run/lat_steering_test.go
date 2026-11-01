// Purpose: Preserve cancellation and steering boundaries around a native LAT review.
// Pattern: Channel-controlled runtime lifecycle tests.
// Usage: go test -race ./internal/agent/run -run 'TestLAT(Cancellation|Steering)'.
// Related: internal/agent/run/lat.go, internal/agent/run/lat_fixture_test.go.
package run

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestLATCancellationStopsBeforeFurtherWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, model, tools, store := latFixture(nativeCall("write_file", map[string]any{"path": "a", "content": "one"}), "Draft")
	model.handlers["review"] = func(ctx context.Context, _ modelapi.Request, _ func(modelapi.Event) error) error {
		cancel()
		return ctx.Err()
	}
	result, err := r.Run(ctx, "Create a", approve, emit)
	if !errors.Is(err, context.Canceled) || result.Status != runrecord.Interrupted || tools.executed != 1 || model.created != model.released {
		t.Fatalf("cancelled review: %+v scopes=%d/%d err=%v", result, model.created, model.released, err)
	}
	last := store.messages[len(store.messages)-1]
	if last.Role != "assistant" || last.Content != "Draft" || last.Incomplete || len(last.ReasoningState) != 0 {
		t.Fatalf("interrupted review lost or altered the completed main draft: %+v", last)
	}
}

func TestLATCancellationDuringPlanStopsParent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, model, tools, store := latFixture("should not run")
	model.handlers["plan"] = func(ctx context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		if err := send(modelapi.Event{Usage: &modelapi.TokenUsage{Input: 20, Output: 3}}); err != nil {
			return err
		}
		cancel()
		return ctx.Err()
	}
	result, err := r.Run(ctx, "Create a", approve, emit)
	if !errors.Is(err, context.Canceled) || result.Status != runrecord.Interrupted || len(model.requests["main"]) != 0 || tools.executed != 0 || model.created != model.released {
		t.Fatalf("cancelled plan continued parent: %+v scopes=%d/%d err=%v", result, model.created, model.released, err)
	}
	if len(tools.views) != 1 || !tools.views[0].closed {
		t.Fatal("cancelled plan retained its read-only executor")
	}
	usageEvents := 0
	var statistics *runrecord.Statistics
	for _, event := range store.events {
		if event.Kind == "usage" && event.Usage != nil && event.Usage.Input == 20 && event.Usage.Output == 3 {
			usageEvents++
		}
		if !event.Internal && event.Checkpoint != nil && event.Checkpoint.Phase == "finished" {
			statistics = event.Statistics
		}
	}
	if usageEvents != 1 || statistics == nil || statistics.Usage == nil || statistics.Usage.Input != 20 || statistics.Usage.Output != 3 {
		t.Fatalf("cancelled phase usage was lost or duplicated: events=%d statistics=%+v", usageEvents, statistics)
	}
}

func TestLATSteeringDiscardsOldFeedback(t *testing.T) {
	r, model, _, store := latFixture("Old draft", "Follow the new request")
	var ready atomic.Bool
	var reviews atomic.Int32
	r.PollSteering = func() []string {
		if ready.Swap(false) {
			return []string{"New user request"}
		}
		return nil
	}
	model.handlers["review"] = func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		if reviews.Add(1) == 1 {
			ready.Store(true)
			return send(modelapi.Event{Delta: "STALE_REVIEW_FINDING", Done: true})
		}
		return send(modelapi.Event{Delta: "NO_FINDINGS", Done: true})
	}
	result, err := r.Run(context.Background(), "Old request", approve, emit)
	if err != nil || result.Text != "Follow the new request" || len(model.requests["review"]) != 2 {
		t.Fatalf("steering: %+v phases=%v err=%v", result, model.scopes, err)
	}
	assertAbsent(t, model.requests["main"][1], "STALE_REVIEW_FINDING")
	found := false
	for _, event := range store.events {
		found = found || event.Kind == "lat_discarded"
	}
	if !found {
		t.Fatal("review discard was not recorded")
	}
}

func TestLATSteeringDuringPlanRestartsPreparation(t *testing.T) {
	r, model, _, store := latFixture("New answer")
	var pending atomic.Bool
	var plans atomic.Int32
	r.PollSteering = func() []string {
		if pending.Swap(false) {
			return []string{"New user request"}
		}
		return nil
	}
	model.handlers["plan"] = func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		if plans.Add(1) == 1 {
			pending.Store(true)
			return send(modelapi.Event{Delta: "STALE_PLAN", Done: true})
		}
		return send(modelapi.Event{Delta: "NEW_PLAN", Done: true})
	}
	result, err := r.Run(context.Background(), "Old request", approve, emit)
	if err != nil || result.Text != "New answer" || model.scopes["plan"] != 2 || model.scopes["critique"] != 1 || model.scopes["refine"] != 1 {
		t.Fatalf("steering did not restart preparation: %+v phases=%v err=%v", result, model.scopes, err)
	}
	assertContains(t, model.requests["critique"][0], "NEW_PLAN", "New user request")
	assertContains(t, model.requests["main"][0], "IMPROVED_PLAN", "New user request")
	assertAbsent(t, model.requests["main"][0], "STALE_PLAN")
	found := false
	for _, event := range store.events {
		found = found || event.Kind == "lat_discarded"
	}
	if !found {
		t.Fatal("superseded plan was not recorded as discarded")
	}
}

func TestLATSteeringCancelsAndJoinsBlockedReview(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, model, _, _ := latFixture("Old draft", "New answer")
		var pending atomic.Bool
		var reviews atomic.Int32
		joined := make(chan struct{})
		r.PollSteering = func() []string {
			if pending.Swap(false) {
				return []string{"A different request"}
			}
			return nil
		}
		model.handlers["review"] = func(ctx context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
			if reviews.Add(1) == 1 {
				pending.Store(true)
				<-ctx.Done()
				close(joined)
				return ctx.Err()
			}
			return send(modelapi.Event{Delta: "NO_FINDINGS", Done: true})
		}
		result, err := r.Run(context.Background(), "Inspect a", approve, emit)
		if err != nil || result.Text != "New answer" || len(model.requests["review"]) != 2 {
			t.Fatalf("steering waited for stale review: %+v %v", result, err)
		}
		select {
		case <-joined:
		default:
			t.Fatal("review worker outlived its steering boundary")
		}
		for _, message := range model.requests["main"][1].Messages {
			if strings.Contains(message.Content, "STALE_REVIEW_FINDING") {
				t.Fatal("stale feedback reached parent after cancellation")
			}
		}
	})
}
