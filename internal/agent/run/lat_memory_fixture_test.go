// Purpose: Project the test journal's original timeline into the independent-memory contract.
// Pattern: In-memory repository fixture.
// Usage: Scripted LAT runtime tests preserve chronological boundaries without main summaries.
// Related: internal/agent/run/lat_test.go, internal/agent/lat/memory/types.go.
package run

import (
	"context"
	"fmt"

	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	"github.com/keonho-kim/meca/internal/storage/history"
)

type journalMemory struct {
	journal *journal
	state   latmemory.State
}

func (m *journalMemory) LATMemory(context.Context, string) (latmemory.State, error) {
	return m.state, nil
}
func (m *journalMemory) InstallLATMemory(_ context.Context, _ string, expected int, next latmemory.State) error {
	if m.state.Revision != expected {
		return fmt.Errorf("stale test memory")
	}
	m.state = next
	return nil
}
func (m *journalMemory) LATWatermark(ctx context.Context, session string, after int64) (latmemory.Mark, error) {
	records, through, err := m.LATRecords(ctx, session, after, int64(len(m.journal.timeline)), latmemory.MaxBytes)
	mark := latmemory.Mark{Through: through}
	for _, record := range records {
		mark.Bytes += int64(len(record.Text))
		if record.Kind == "user" {
			mark.LatestUser = record.Sequence
		}
	}
	return mark, err
}
func (m *journalMemory) LATRecords(_ context.Context, _ string, after, through int64, budget int) ([]latmemory.Record, int64, error) {
	var records []latmemory.Record
	for _, source := range m.journal.timeline {
		if source.Sequence <= after || source.Sequence > through {
			continue
		}
		if source.Kind == "message" && source.Message.Role != "user" {
			continue
		}
		if source.Kind == "event" && source.Event.Operation == nil {
			continue
		}
		projected, err := history.Compaction([]history.Record{source})
		if err != nil {
			return nil, after, err
		}
		for _, value := range projected {
			records = append(records, latmemory.Record{Sequence: value.Sequence, Kind: value.Role, Text: value.Text})
		}
	}
	return records, through, nil
}
