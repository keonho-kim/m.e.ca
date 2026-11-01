// Purpose: Verify ephemeral LAT decisions perform no checkpoint I/O while final state remains durable.
// Pattern: Journal boundary and cancellation regression.
// Usage: Run TestForkCheckpoint with the runtime tests.
// Related: internal/agent/run/journal.go, internal/agent/lat/transcript/audit.go.
package run

import (
	"context"
	"errors"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestForkCheckpointSkipsIntermediateIOAndCommitsFinalState(t *testing.T) {
	j := &journal{}
	s := execution{runner: Runner{fork: &forkConfig{}, Journal: j, Session: "s"}, checkpoint: runrecord.Checkpoint{ID: "child_run", Status: runrecord.Running, Phase: "decision", InstructionsDigest: strings.Repeat("a", 64)}}
	for range 20 {
		if err := s.persist(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(j.events) != 0 {
		t.Fatal("ephemeral decisions reached the journal")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.persist(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("skipped checkpoint ignored cancellation", err)
	}
	s.checkpoint.Phase, s.checkpoint.Status = "finished", runrecord.Answered
	if err := s.persist(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(j.events) != 1 || j.events[0].Checkpoint.Phase != "finished" {
		t.Fatal("final status was not committed")
	}
}
