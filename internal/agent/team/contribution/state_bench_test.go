// Purpose: Measure already-reviewed path checks and unchanged reclamation watermarks.
// Pattern: Pure-state steady-state benchmark.
// Usage: Compare lookup and pruning cost with a slow teammate retaining path facts.
// Related: internal/agent/team/contribution/state.go.
package contribution

import (
	"fmt"
	"testing"
)

func BenchmarkReviewedChanges(b *testing.B) {
	var s State
	setScope(&s, "Ren", "Check integration", []string{"/project"})
	for i := range 4096 {
		s.Record("Hana", []string{fmt.Sprintf("/project/file%d", i)}, 1)
	}
	b.Run("check", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if err := s.Check("Ren", []string{"/project/result"}, 1); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("prune", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			s.Prune(0)
		}
	})
}
