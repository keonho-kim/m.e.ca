// Purpose: Read original tool arguments or results without hydrating unrelated large payloads.
// Pattern: On-demand repository projection.
// Usage: Terminal and server detail readers select bounded originals outside their state locks.
// Related: internal/storage/sqlite/team_references.go, internal/commands/server/detail.go, internal/tui/session/tool_details.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	json "github.com/goccy/go-json"

	"io/fs"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) ToolEvent(ctx context.Context, ref runrecord.EventReference, bounds history.ReadBounds) (runrecord.Event, error) {
	return s.toolDetail(ctx, ref, bounds, false)
}

// ToolRequest selects the original arguments without hydrating the result body.
func (s *Store) ToolRequest(ctx context.Context, ref runrecord.EventReference, bounds history.ReadBounds) (runrecord.Event, error) {
	return s.toolDetail(ctx, ref, bounds, true)
}

func (s *Store) toolDetail(ctx context.Context, ref runrecord.EventReference, bounds history.ReadBounds, requestOnly bool) (result runrecord.Event, resultErr error) {
	if ref.Sequence < 1 {
		return result, faults.Errorf(diagcatalog.SqliteInvalidToolDetailReference)
	}
	source := s
	page := teamPage{parent: s}
	defer func() { resultErr = errors.Join(resultErr, page.close()) }()
	if ref.Session != s.session.ID {
		var err error
		source, err = page.source(ctx, ref.Session)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return result, errors.Join(history.ErrNotFound, err)
			}
			return result, err
		}
	}
	var record storedRecord
	var arguments []byte
	record.kind = "event"
	maxBytes := bounds.MaxBytes
	if maxBytes == 0 {
		maxBytes = content.MaxBytes
	}
	err := source.db.QueryRowContext(ctx, "SELECT CASE WHEN length(body)<=? THEN body END,refs,raw_bytes,display_args FROM tool_events WHERE sequence=?", maxBytes, ref.Sequence).Scan(&record.body, &record.refs, &record.size, &arguments)
	if errors.Is(err, sql.ErrNoRows) {
		return result, history.ErrNotFound
	}
	if err != nil {
		return result, err
	}
	if record.body == nil {
		return result, history.ErrReadLimit
	}
	budget := newReadBudget(bounds.MaxBytes, bounds.Reserve)
	if err := budget.take(len(record.body)); err != nil {
		return result, err
	}
	payloads := pagePayloads{store: source, ctx: ctx, budget: budget}
	fields := history.DisplayFields("event")
	if requestOnly {
		fields = map[string]bool{"call": true, "kind": true, "agent": true, "run_id": true, "call_id": true, "occurred_at": true}
	}
	body, err := payloads.unpack(0, record, fields)
	if err != nil {
		return result, err
	}
	if requestOnly {
		err := json.Unmarshal(body, &result)
		return result, err
	}
	projected, err := history.Display(0, "event", body, record.body, arguments)
	return projected.Event, err
}
