// Purpose: Keep stable LAT layout free of repeated group-identity allocations.
// Pattern: Warm presentation benchmark.
// Usage: Run BenchmarkLATLayout with -benchmem.
// Related: internal/tui/session/lat_output.go, internal/tui/session/chat_projection.go.
package session

import (
	"fmt"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func BenchmarkLATLayout(b *testing.B) {
	for _, enabled := range []bool{false, true} {
		b.Run(fmt.Sprint(enabled), func(b *testing.B) {
			a := screenFixture(b)
			a.worker = workerAgent
			a.stream = "A stable response."
			if enabled {
				a.latOutput = latOutput{group: transcript.LATGroup(runrecord.LATRun{ID: "lat_review", ParentRunID: "run_main_123456789", Phase: "review"}), running: true}
			}
			a.chatLayout(80)
			b.ReportAllocs()
			for b.Loop() {
				a.chatLayout(80)
			}
		})
	}
}
