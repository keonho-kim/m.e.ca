// Purpose: Export hydrated original records in bounded pages without retaining the whole session.
// Pattern: Persistence export adapter.
// Usage: meca --export reconstructs payloads only after each metadata page releases its DB connection.
// Related: internal/storage/sqlite/record_page.go, internal/storage/sqlite/record_codec.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"

	"io"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

func (s *Store) Export(ctx context.Context, session string, output io.Writer) (resultErr error) {
	if err := s.checkSession(session); err != nil {
		return err
	}
	pageTeam := teamPage{parent: s}
	defer func() { resultErr = errors.Join(resultErr, pageTeam.close()) }()
	encoder := json.NewEncoder(output)
	var after int64
	for {
		page, err := s.recordPage(ctx, "SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes) FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence WHERE t.sequence>? ORDER BY t.sequence LIMIT 64", after)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		payloads := pagePayloads{store: s, ctx: ctx}
		pageTeam.payloads = &payloads
		if err := pageTeam.prepare(ctx, page); err != nil {
			return err
		}
		for i, stored := range page {
			if err := payloads.include(i, s, stored.refs); err != nil {
				return err
			}
		}
		for i, stored := range page {
			payloads.store = s
			body, err := payloads.unpack(i, stored, nil)
			if err != nil {
				return err
			}
			if stored.kind == "event" {
				var event runrecord.Event
				if err := json.Unmarshal(body, &event); err != nil {
					return err
				}
				if event.Kind == "team_reference" {
					pageTeam.index = i
					event, err = pageTeam.resolve(ctx, event, false)
					if err != nil {
						return err
					}
					body, err = json.Marshal(event)
					if err != nil {
						return err
					}
				}
				if event.Kind == "run_checkpoint" && event.Checkpoint != nil {
					full, err := s.runState(ctx, session, event.Checkpoint.ID)
					if err != nil {
						return err
					}
					body, err = json.Marshal(full)
					if err != nil {
						return err
					}
				}
			}
			if len(body) > recordcontent.MaxBytes || !json.Valid(body) {
				return faults.Errorf(diagcatalog.SqliteInvalidSessionExportRecord)
			}
			record := struct {
				Sequence int64           `json:"sequence"`
				Kind     string          `json:"kind"`
				Body     json.RawMessage `json:"body"`
			}{stored.sequence, stored.kind, body}
			if err := encoder.Encode(record); err != nil {
				return err
			}
			after = stored.sequence
		}
	}
}
