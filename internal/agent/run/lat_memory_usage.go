// Purpose: Commit background measurements on the runtime owner without resetting foreground usage.
// Pattern: Serial accounting projection from joined-worker receipts.
// Usage: LAT-memory polling feeds real totals, attempt provenance and progress to the existing UI.
// Related: internal/agent/run/usage.go, internal/tui/session/tokens.go.
package run

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/agent/inference"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (s *execution) recordLATMemoryUsage(ctx context.Context, update latmemory.Update) error {
	for _, receipt := range update.Receipts {
		s.statistics.add(receipt.Usage, receipt.Measured)
		generation := runrecord.Generation{ID: receipt.ID, Role: inference.LATCompaction, Attempt: receipt.Attempt, ElapsedMillis: receipt.Elapsed.Milliseconds()}
		var rate *float64
		if receipt.Usage != nil {
			rate = outputRate(receipt.Usage.Output, receipt.Measured)
		}
		if err := s.recordUsage(ctx, receipt.Usage, rate, generation); err != nil {
			return errors.Join(errRuntimeRecord, err)
		}
	}
	if progress := update.Progress; progress != nil && progress.Usage != nil {
		generation := runrecord.Generation{ID: progress.ID, Role: inference.LATCompaction, Attempt: progress.Attempt}
		if err := s.emit(runrecord.Event{Kind: "usage_progress", Usage: progress.Usage, Generation: &generation}); err != nil {
			return err
		}
	}
	s.checkpoint.ModelCalls = s.usage.snapshot().ModelCalls
	return nil
}
