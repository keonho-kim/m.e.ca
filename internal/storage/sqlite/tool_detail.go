// Purpose: Read one original tool result for user-requested expansion of a compact transcript block.
// Pattern: On-demand repository projection.
// Usage: The TUI detail worker calls ToolEvent outside the input event loop.
// Related: internal/storage/sqlite/team_references.go, internal/tui/session/tool_details.go.
package sqlite

import (
	"context"
	"errors"
	"fmt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) ToolEvent(ctx context.Context, ref runrecord.EventReference) (result runrecord.Event, resultErr error) {
	if ref.Sequence < 1 {
		return result, fmt.Errorf("invalid tool detail reference")
	}
	source := s
	page := teamPage{parent: s}
	defer func() { resultErr = errors.Join(resultErr, page.close()) }()
	if ref.Session != s.session.ID {
		var err error
		source, err = page.source(ctx, ref.Session)
		if err != nil {
			return result, err
		}
	}
	var record storedRecord
	var arguments []byte
	record.kind = "event"
	err := source.db.QueryRowContext(ctx, "SELECT body,refs,raw_bytes,display_args FROM tool_events WHERE session_id=? AND sequence=?", ref.Session, ref.Sequence).Scan(&record.body, &record.refs, &record.size, &arguments)
	if err != nil {
		return result, err
	}
	body, err := source.unpackRecord(ctx, record, history.DisplayFields("event"))
	if err != nil {
		return result, err
	}
	projected, err := history.Display(0, "event", body, record.body, arguments)
	return projected.Event, err
}
