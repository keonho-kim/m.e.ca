// Purpose: Verify run recovery and review evidence remain durable without replaying interrupted work.
// Pattern: Repository integration tests.
// Usage: go test ./internal/storage/sqlite -run TestReviewAndMonitorRoundTrip.
// Related: internal/storage/sqlite/monitor.go, internal/storage/sqlite/memory_sources.go, internal/storage/recovery/recovery.go.
package sqlite

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/conclave"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/storage/recovery"
	"strings"
	"testing"
)

func TestRunRecoveryPreservesReviewAndMonitor(t *testing.T) {
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
	state := run.Checkpoint{ID: "r", Objective: "fix", Status: run.Running, Phase: "decision", InstructionsDigest: strings.Repeat("a", 64)}
	review := conclave.Result{CallID: "c", CandidateDigest: strings.Repeat("a", 64), Lines: 1, Attempts: 1, Complete: true, Verdict: conclave.Verdict{Decision: "reject", Findings: []conclave.Finding{{Path: "a", Side: "request", Reason: "missing test"}}}}
	for _, event := range []run.Event{{Kind: "run_checkpoint", Checkpoint: &state}, {Kind: "conclave", Conclave: &review}, {Kind: "monitor", Monitor: &monitor.Record{Sequence: 1, RunID: "t", Kind: "review_rejected", Detail: "missing test"}}} {
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
	restored, err := store.LatestRun(ctx, session.ID)
	if err != nil || restored.Status != "interrupted" {
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
	if len(records) != 1 || records[0].Role != "failure" || !strings.Contains(records[0].Text, "interrupted") {
		t.Fatalf("compaction retained duplicate audit records: %+v", records)
	}
}
