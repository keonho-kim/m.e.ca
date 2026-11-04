// Purpose: Bound empty instruction checks independently of batch width and token delivery.
// Pattern: Observable admission-count regressions and microbenchmarks.
// Usage: Runtime tests run independent temporary file calls and empty input boundaries.
// Related: internal/agent/run/steering.go, internal/agent/run/batch_wave.go.
package run

import (
	"context"
	"fmt"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/tool"
)

func TestAutomaticBatchInstructionChecksDoNotScaleWithFileCount(t *testing.T) {
	counts := make([]int, 0, 2)
	for _, count := range []int{1, 16} {
		calls := make([]tool.Call, 0, count)
		for i := range count {
			path := fmt.Sprint(i)
			calls = append(calls, batchCall(path, "write_file", map[string]any{"path": path, "content": "content"}))
		}
		r, _, _, _, _ := realBatchRunner(t, calls)
		polls := 0
		r.PollSteering = func() []string { polls++; return nil }
		if _, err := r.Run(context.Background(), "Create independent files", approve, emit); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, polls)
	}
	if counts[1] > counts[0]+1 {
		t.Fatal("automatic batch checks grew with file count", counts)
	}
}

func BenchmarkEmptyInstructionCheck(b *testing.B) {
	for _, enabled := range []bool{false, true} {
		b.Run(fmt.Sprint(enabled), func(b *testing.B) {
			s := execution{}
			if enabled {
				s.runner.PollSteering = func() []string { return nil }
			}
			b.ReportAllocs()
			for b.Loop() {
				if err := s.checkInstructions(context.Background(), 0); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
