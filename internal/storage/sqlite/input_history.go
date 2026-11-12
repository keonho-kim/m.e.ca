// Purpose: Restore complete user input text within a bounded recall budget without images or tool output.
// Pattern: Consumer-specific repository projection.
// Usage: Terminal restoration requests up to 200 whole inputs within the shared 16MiB recall policy.
// Related: internal/tui/session/session.go, internal/storage/sqlite/record_codec.go.
package sqlite

import (
	"context"

	"slices"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/processmemory"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"

	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) RecentInputs(ctx context.Context, session string, limit int) ([]string, error) {
	if err := s.checkSession(session); err != nil {
		return nil, err
	}
	if limit < 1 || limit > history.MaxRecentInputs {
		return nil, faults.Errorf(diagcatalog.SqliteInputHistoryLimitMustBeBetweenAnd)
	}
	// Admit only selected immutable envelopes, then grow for decoded text.
	// The streaming decoder also owns its temporary scratch reservation.
	memory := processmemory.NewPayloadLease()
	defer memory.Close()
	page, metadata, err := s.recentInputPage(ctx, limit, memory.Resize)
	if err != nil {
		return nil, err
	}
	var inputs []string
	bytes := 0
	for _, record := range page {
		remaining := history.RecentInputBytes - bytes
		if remaining == 0 {
			break
		}
		text, clipped, err := recordcontent.TextPrefix(record.body, record.refs, func(id string, limit int) (string, bool, error) {
			budget := newReadBudget(0, func(selected int) error { return memory.Resize(metadata + 3*bytes + 4*selected) })
			return s.readTextPrefix(ctx, id, budget, limit)
		}, remaining)
		if err != nil {
			return nil, err
		}
		if clipped {
			if len(inputs) == 0 {
				continue
			}
			break
		}
		bytes += len(text)
		inputs = append(inputs, text)
		if err := memory.Resize(metadata + 3*bytes); err != nil {
			return nil, err
		}
	}
	slices.Reverse(inputs)
	return inputs, nil
}
