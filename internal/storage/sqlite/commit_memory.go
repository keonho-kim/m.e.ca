// Purpose: Admit large journal transformations before allocating serialized record copies.
// Pattern: Persistence-owned transient-memory estimate.
// Usage: Commit holds this reservation through preparation, encoding and transaction completion.
// Related: internal/storage/sqlite/commit.go, internal/processmemory/transient.go.
package sqlite

import (
	"context"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/processmemory"
)

func reserveJournal(ctx context.Context, records []runrecord.Entry) (func(), error) {
	var bytes int64
	for _, record := range records {
		if m := record.Message; m != nil {
			bytes += int64(m.ContextBytes())*3 + 1024
		}
		if e := record.Event; e != nil {
			bytes += int64(len(e.Text)+len(e.Thinking)+len(e.Output)+len(e.Capture)+len(e.ChangePreview))*3 + 1024
			if e.Call != nil {
				bytes += int64(len(e.Call.Arguments)) * 3
			}
			if c := e.Checkpoint; c != nil {
				bytes += int64(len(c.Objective)+len(c.Observation)+len(c.Blocker)+len(c.LastChange)+len(c.ReportReason)) * 3
				for _, f := range c.Work.Files {
					bytes += int64(len(f.Path)+len(f.Version)+len(f.ObservedRun)+len(f.SourceID)) + 256
				}
			}
		}
	}
	if bytes <= 64<<10 {
		return func() {}, ctx.Err()
	}
	return processmemory.AcquirePayload(ctx, bytes)
}
