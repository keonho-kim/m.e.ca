// Purpose: Verify Goal recovery and review evidence remain durable without replaying interrupted work.
// Pattern: Repository integration tests.
// Usage: go test ./internal/storage/sqlite -run TestGoalRecovery.
// Related: internal/storage/sqlite/monitor.go, internal/storage/sqlite/memory_sources.go, internal/storage/recovery/recovery.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/goal"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/storage/recovery"
	"strings"
	"testing"
)

func TestGoalRecoveryPreservesReviewAndMonitor(t *testing.T) {
	ctx := context.Background()
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	session := store.Session()
	state := goal.State{ID: "g", Request: "fix", Objective: "fix", Status: "running", Tasks: []goal.Task{{ID: "t", Objective: "fix", Kind: "work", Status: "running"}}}
	review := conclave.Result{CallID: "c", CandidateDigest: strings.Repeat("a", 64), Lines: 1, Reviews: 3, Required: 2, Complete: true, Votes: []conclave.Vote{{Index: 1, Reason: "missing test"}, {Index: 2, Reason: "missing test"}, {Index: 3, Reason: "missing test"}}}
	for _, event := range []run.Event{{Kind: "goal_state", Goal: &state, Internal: true}, {Kind: "conclave", Conclave: &review}, {Kind: "monitor", Monitor: &monitor.Record{Sequence: 1, GoalID: "g", TaskID: "t", Kind: "review_rejected", Detail: "missing test"}}} {
		if err := store.RecordEvent(ctx, session.ID, event); err != nil {
			t.Fatal(err)
		}
	}
	if err := recovery.Restore(ctx, store, session.ID); err != nil {
		t.Fatal(err)
	}
	if err := recovery.Restore(ctx, store, session.ID); err != nil {
		t.Fatal(err)
	}
	restored, err := store.LatestGoal(ctx, session.ID)
	if err != nil || restored.Status != "interrupted" || restored.Tasks[0].Status != "interrupted" {
		t.Fatalf("%+v %v", restored, err)
	}
	history, err := store.MonitorHistory(ctx, session.ID)
	if err != nil || len(history) != 1 || history[0].Detail != "missing test" {
		t.Fatalf("%+v %v", history, err)
	}
	records, _, err := store.MemoryRecords(ctx, session.ID, 0, 1<<60)
	if err != nil {
		t.Fatal(err)
	}
	goals, reviews, observations := 0, 0, 0
	for _, record := range records {
		if record.Goal != nil {
			goals++
		}
		if record.Review != nil {
			reviews++
		}
		if record.Role == "monitor" {
			observations++
		}
	}
	if goals != 2 || reviews != 1 || observations != 1 {
		t.Fatalf("lost or duplicated source facts: %d/%d/%d", goals, reviews, observations)
	}
}
