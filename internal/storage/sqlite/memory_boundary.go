// Purpose: Retain complete recent native message groups while selecting an older compaction prefix.
// Pattern: Paged repository projection.
// Usage: Compaction reads metadata pages and hydrates only the recent messages needed for its tail.
// Related: internal/storage/history/boundary.go, internal/storage/sqlite/record_page.go.
package sqlite

import (
	"context"

	json "github.com/goccy/go-json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) CompactionBoundary(ctx context.Context, session string, through int64, keepBytes int) (boundary int64, err error) {
	if err := s.checkSession(session); err != nil {
		return 0, err
	}
	if through < 0 || keepBytes < 0 {
		return 0, faults.Errorf(diagcatalog.SqliteInvalidCompactionTailBounds)
	}
	completed, err := s.lastCompletedDecision(ctx, through)
	if err != nil {
		return 0, err
	}
	pin, pinErr := s.unreadObservationBoundary(ctx, through, completed)
	if pinErr != nil {
		return 0, pinErr
	}
	defer func() {
		if err == nil {
			boundary = min(boundary, pin)
		}
	}()
	if keepBytes == 0 {
		return through, nil
	}
	selection := history.TailSelection{Target: keepBytes}
	fields := map[string]bool{"role": true, "calls": true, "tool_call_id": true, "tool_name": true, "incomplete": true}
	before := through + 1
	for {
		page, err := s.decisionPage(ctx, completed, before, true)
		if err != nil {
			return 0, err
		}
		if len(page) == 0 {
			return 0, nil
		}
		for _, record := range page {
			body, err := s.unpackRecord(ctx, record, fields)
			if err != nil {
				return 0, err
			}
			var message modelapi.Message
			if err := json.Unmarshal(body, &message); err != nil {
				return 0, err
			}
			if message.ToolName == "edit_file" {
				body, err := s.unpackRecord(ctx, record, map[string]bool{"content": true})
				if err != nil {
					return 0, err
				}
				if err := json.Unmarshal(body, &message); err != nil {
					return 0, err
				}
			}
			if boundary, ready := selection.Keep(record.sequence, record.size, message); ready {
				return boundary, nil
			}
			before = record.sequence
		}
	}
}
