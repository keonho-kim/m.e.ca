// Purpose: Page ordered message and tool-event references for terminal history without duplicating bodies.
// Pattern: Repository projection.
// Usage: Used by session restoration and scrolling into older history.
// Related: internal/storage/sqlite/timeline_test.go, internal/tui/session/history.go.
package sqlite

import (
	"context"
	"errors"
	"fmt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"

	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Store) Timeline(ctx context.Context, session string, before int64, limit int) (records []history.Record, cursor int64, err error) {
	return s.timeline(ctx, session, before, limit, false)
}

// TimelineSummaries leaves original tool output in storage until the user expands it.
func (s *Store) TimelineSummaries(ctx context.Context, session string, before int64, limit int) ([]history.Record, int64, error) {
	return s.timeline(ctx, session, before, limit, true)
}
func (s *Store) timeline(ctx context.Context, session string, before int64, limit int, compact bool) (records []history.Record, cursor int64, err error) {
	if limit < 1 || limit > 200 {
		return nil, 0, fmt.Errorf("invalid history page")
	}
	if before == 0 {
		before = 1 << 62
	}
	page, err := s.displayPage(ctx, session, before, limit)
	if err != nil {
		return nil, 0, err
	}
	payloads := pagePayloads{store: s, ctx: ctx}
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
		if err := payloads.include(i, entry.record.refs); err != nil {
			return nil, 0, err
		}
	}
	for i, entry := range page {
		record := entry.record
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
		payloads.store = s
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
