// Purpose: Project original user instructions and actual outcomes for independent review memory.
// Pattern: Paged repository projection excluding assistant claims and main summaries.
// Usage: LAT preparation reads bounded pages after the last independently covered boundary.
// Related: internal/storage/sqlite/lat_memory.go, internal/storage/sqlite/events.go.
package sqlite

import (
	"context"
	"errors"
	"fmt"

	json "github.com/goccy/go-json"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
)

func (s *Store) LATWatermark(ctx context.Context, session string, after int64) (latmemory.Mark, error) {
	var mark latmemory.Mark
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(t.sequence),?2),COALESCE(SUM(CASE WHEN m.role='user' THEN m.content_bytes ELSE e.review_bytes END),0),COALESCE(MAX(CASE WHEN m.role='user' THEN t.sequence ELSE 0 END),0)
FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence
LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence
WHERE t.session_id=?1 AND t.sequence>?2 AND (m.role='user' OR e.review_bytes>0)`, session, after).Scan(&mark.Through, &mark.Bytes, &mark.LatestUser)
	return mark, err
}

func (s *Store) LATRecords(ctx context.Context, session string, after, through int64, budget int) ([]latmemory.Record, int64, error) {
	if budget < 1 || budget > latmemory.MaxBytes {
		return nil, after, fmt.Errorf("invalid LAT source input budget")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.content_bytes,e.review_bytes),COALESCE(e.memory_role,'user'),COALESCE(substr(e.memory_text,1,?4),'')
FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence
LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence
WHERE t.session_id=?1 AND t.sequence>?2 AND t.sequence<=?3 AND (m.role='user' OR e.review_bytes>0) ORDER BY t.sequence LIMIT 64`, session, after, through, latmemory.TargetBytes+1)
	if err != nil {
		return nil, after, err
	}
	type source struct {
		record     storedRecord
		kind, text string
	}
	var page []source
	bytes := 0
	for rows.Next() {
		var entry source
		r := &entry.record
		if err = rows.Scan(&r.sequence, &r.kind, &r.body, &r.refs, &r.size, &entry.kind, &entry.text); err != nil {
			break
		}
		if bytes+r.size > budget && len(page) > 0 {
			break
		}
		if r.size > budget {
			err = latmemory.ErrSourceBudget
			break
		}
		bytes += r.size
		page = append(page, entry)
	}
	if err = errors.Join(err, rows.Err(), rows.Close()); err != nil {
		return nil, after, err
	}
	if len(page) == 0 {
		return nil, through, nil
	}
	out := make([]latmemory.Record, 0, len(page))
	for _, entry := range page {
		if entry.record.kind == "event" {
			text, err := s.latObservation(ctx, entry.record, entry.text)
			if err != nil {
				return nil, after, err
			}
			out = append(out, latmemory.Record{Sequence: entry.record.sequence, Kind: entry.kind, Text: text})
			continue
		}
		if entry.record.kind == "message" && entry.record.size > latmemory.MaxBytes {
			return nil, after, fmt.Errorf("LAT original user record exceeds the 16 MiB memory input envelope; original retained")
		}
		fields := map[string]bool{"content": true}
		body, err := s.unpackRecord(ctx, entry.record, fields)
		if err != nil {
			return nil, after, err
		}
		var value struct{ Content string }
		if err = json.Unmarshal(body, &value); err != nil {
			return nil, after, err
		}
		out = append(out, latmemory.Record{Sequence: entry.record.sequence, Kind: entry.kind, Text: value.Content})
	}
	return out, out[len(out)-1].Sequence, nil
}
