// Purpose: Attach request provenance and committed references at the ordered event publication boundary.
// Pattern: Request-owned event adapter.
// Usage: Runtime startup binds this emitter once; successful delivery then emits central diagnostics.
// Related: internal/agent/run/start.go, internal/agent/run/logging.go, internal/agent/run/record.go.
package run

import (
	"context"
	"errors"
	"time"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func newRunEmitter(ctx context.Context, journal Journal, session, runID string, send func(runrecord.Event) error) func(runrecord.Event) error {
	return func(event runrecord.Event) error {
		if event.RunID == "" {
			event.RunID = runID
		}
		if event.OccurredAt == "" {
			event.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if references, ok := journal.(interface {
			LatestEventReference(context.Context) (*runrecord.EventReference, error)
		}); ok {
			switch event.Kind {
			case "tool_succeeded", "tool_failed", "tool_interrupted", "command_finished", "denied", "skill_loaded":
				flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				reference, err := references.LatestEventReference(flush)
				cancel()
				if err != nil {
					return errors.Join(errRuntimeRecord, err)
				}
				event.Reference = reference
			}
		}
		if err := send(event); err != nil {
			if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
				return ctx.Err()
			}
			return errors.Join(errRuntimeRecord, err)
		}
		logEvent(ctx, event, session)
		return nil
	}
}
