// Purpose: Read visible transcript pages and their durable boundary without duplicating stored bodies.
// Pattern: Repository projection.
// Usage: Session restoration and server history handoffs share this storage projection.
// Related: internal/storage/sqlite/timeline_test.go, internal/tui/session/history.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/media"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/content"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/history"
)

// TimelineHead reads the committed visible transcript boundary without restoring bodies.
func (s *Store) TimelineHead(ctx context.Context, session string) (int64, error) {
	if err := s.checkSession(session); err != nil {
		return 0, err
	}
	var sequence int64
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE((SELECT t.sequence "+visibleHistoryJoin+" WHERE "+visibleHistoryPredicate+" ORDER BY t.sequence DESC LIMIT 1),0)").Scan(&sequence)
	return sequence, err
}

func (s *Store) Timeline(ctx context.Context, session string, before int64, limit int) (records []history.Record, cursor int64, err error) {
	if err := s.checkSession(session); err != nil {
		return nil, 0, err
	}
	return s.timeline(ctx, session, before, limit, false, history.ReadBounds{})
}

// TimelineSummaries leaves original tool output in storage until the user expands it.
func (s *Store) TimelineSummaries(ctx context.Context, session string, before int64, limit int, bounds history.ReadBounds) ([]history.Record, int64, error) {
	if err := s.checkSession(session); err != nil {
		return nil, 0, err
	}
	return s.timeline(ctx, session, before, limit, true, bounds)
}
func (s *Store) timeline(ctx context.Context, session string, before int64, limit int, compact bool, bounds history.ReadBounds) (records []history.Record, cursor int64, err error) {
	if limit < 1 || limit > 200 || bounds.MaxBytes < 0 || bounds.TextBytes < 0 {
		return nil, 0, faults.Errorf(diagcatalog.SqliteInvalidHistoryPage)
	}
	if before == 0 {
		before = 1 << 62
	}
	page, err := s.displayPage(ctx, session, before, limit, bounds.MaxBytes)
	if err != nil {
		return nil, 0, err
	}
	budget := newReadBudget(bounds.MaxBytes, bounds.Reserve)
	payloads := pagePayloads{store: s, ctx: ctx, budget: budget}
	pageTeam := teamPage{parent: s, compact: compact, payloads: &payloads}
	var metadata []storedRecord
	for i, entry := range page {
		if entry.eventKind == "team_reference" {
			if metadata == nil {
				metadata = make([]storedRecord, len(page))
			}
			metadata[i] = entry.record
		}
	}
	if err := pageTeam.prepare(ctx, metadata); err != nil {
		_ = pageTeam.close()
		return nil, 0, err
	}
	defer func() { err = errors.Join(err, pageTeam.close()) }()
	bytes := 0
	for i, entry := range page {
		if err := payloads.include(i, s, entry.record.refs); err != nil {
			return nil, 0, err
		}
	}
	for i, entry := range page {
		record := entry.record
		payloads.store = s
		if err := budget.take(len(record.body)); err != nil {
			if len(records) > 0 {
				break
			}
			return nil, 0, err
		}
		if bounds.TextBytes > 0 && record.kind == "message" {
			payloads.index = i
			var visible history.Record
			visible.Sequence, visible.Kind = record.sequence, record.kind
			var envelope struct {
				ID, Role   string
				Incomplete bool
				Images     []struct {
					Path, MIME, Digest string
					Width, Height      int
					Data               json.RawMessage
				}
			}
			if err := json.Unmarshal(record.body, &envelope); err != nil {
				return nil, 0, err
			}
			visible.Message.ID, visible.Message.Role, visible.Message.Incomplete = envelope.ID, envelope.Role, envelope.Incomplete
			for _, image := range envelope.Images {
				var imageBytes int64
				if raw := image.Data; len(raw) >= 2 && raw[0] == '"' {
					imageBytes = int64((len(raw) - 2) / 4 * 3)
					if len(raw) > 3 && raw[len(raw)-2] == '=' {
						imageBytes--
					}
					if len(raw) > 4 && raw[len(raw)-3] == '=' {
						imageBytes--
					}
				}
				visible.Images = append(visible.Images, media.Metadata{Path: image.Path, MIME: image.MIME, Digest: image.Digest, Width: image.Width, Height: image.Height, Bytes: imageBytes})
			}
			visible.Message.Content, visible.Truncated, err = content.TextPrefix(record.body, record.refs, func(id string, limit int) (string, bool, error) {
				return payloads.prefix(id, limit)
			}, bounds.TextBytes)
			if err != nil {
				return nil, 0, err
			}
			if err := visible.Message.Check(); err != nil {
				return nil, 0, err
			}
			records = append(records, visible)
			cursor = record.sequence
			continue
		}
		// Page acceptance precedes payload hydration. An individually large
		// visible record remains readable, but cannot overfill a preceding page.
		if !compact && bytes+record.size > 16*1024*1024 && len(records) > 0 {
			break
		}
		fields := history.DisplayFields(record.kind)
		tool := terminalTool(entry.eventKind)
		if compact && tool {
			fields = compactToolFields(fields)
		}
		body, decodeErr := payloads.unpack(i, record, fields)
		if decodeErr != nil {
			return nil, 0, decodeErr
		}
		bytes += len(body)
		if bytes > 16*1024*1024 && len(records) > 0 {
			break
		}
		r, err := history.Display(record.sequence, record.kind, body, record.body, entry.arguments)
		if err != nil {
			return nil, 0, err
		}
		if r.Kind == "event" && r.Event.Kind == "team_reference" {
			pageTeam.index = i
			r.Event, err = pageTeam.resolve(ctx, r.Event, true)
			if err != nil {
				return nil, 0, err
			}

			extra := len(r.Event.Text) + len(r.Event.Output) + len(r.Event.ChangePreview)
			if r.Event.Call != nil {
				extra += len(r.Event.Call.Arguments)
			}
			if bytes+extra > 16*1024*1024 && len(records) > 0 {
				break
			}
			bytes += extra
		}
		if compact && tool {
			r.Event.Reference = &runrecord.EventReference{Session: s.session.ID, Sequence: entry.source}
		}
		if compact && r.Event.PublicKind() == "skill_loaded" && r.Event.Reference == nil {
			r.Event.Reference = &runrecord.EventReference{Session: s.session.ID, Sequence: entry.source}
		}
		cursor = record.sequence
		records = append(records, r)
	}
	for l, r := 0, len(records)-1; l < r; l, r = l+1, r-1 {
		records[l], records[r] = records[r], records[l]
	}
	return records, cursor, nil
}

func terminalTool(kind string) bool {
	switch kind {
	case "tool_succeeded", "tool_failed", "tool_interrupted", "command_finished", "denied":
		return true
	}
	return false
}
