// Purpose: Measure repeated checkpoints with a large unchanged objective.
// Pattern: Persistence hot-path benchmark.
// Usage: go test ./internal/storage/sqlite -run '^$' -bench BenchmarkRunStatePreparation -benchmem.
// Related: internal/storage/sqlite/run_state.go, internal/storage/sqlite/prepared_payload.go.
package sqlite

import (
	"path/filepath"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func BenchmarkRunStatePreparation(b *testing.B) {
	sessions, err := OpenSessions(filepath.Join(b.TempDir(), "sessions"))
	if err != nil {
		b.Fatal(err)
	}
	project := b.TempDir()
	s, err := sessions.Create(b.Context(), project, project)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	cp := runrecord.Checkpoint{ID: "run_benchmark", Objective: strings.Repeat("Review this source and preserve its behavior.\n", 100000), Status: runrecord.Running, Phase: "decision"}
	event := runrecord.Event{Kind: "run_checkpoint", Checkpoint: &cp}
	if err := s.RecordEvent(b.Context(), s.Session().ID, event); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp.ModelCalls = i
		encoder := journalEncoder{store: s, ctx: b.Context(), lookup: s.db}
		if _, err := encoder.prepareRunState(event); err != nil {
			b.Fatal(err)
		}
	}
}
