// Purpose: Project consumed browser results before loading original observations from SQLite.
// Pattern: Repository read projection with a durable assistant acknowledgement.
// Usage: Decision history, compaction tail selection and summary sources share the classification.
// Related: internal/storage/sqlite/context.go, internal/storage/sqlite/memory_sources.go.
package sqlite

import (
	"context"
	"database/sql"
	"errors"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

// Existing observations have either an image or a result ID in their inline
// content. A packed observation records the same fact in its metadata.
const browserObservationSQL = `CASE WHEN m.role='tool' AND json_extract(m.body,'$.tool_name')='` + toolapi.BrowserInspect + `' THEN
 CASE WHEN json_extract(m.body,'$.transient_observation')=1
        OR json_array_length(m.body,'$.images')>0 OR json_type(m.body,'$.images')='null' THEN 1
      WHEN json_valid(json_extract(m.body,'$.content'))
      THEN COALESCE(length(json_extract(json_extract(m.body,'$.content'),'$.result_id'))>0,0)
      ELSE 0 END
 ELSE 0 END`

func (s *Store) lastCompletedDecision(ctx context.Context, through int64) (int64, error) {
	var sequence int64
	err := s.db.QueryRowContext(ctx, `
 SELECT t.sequence FROM timeline t JOIN messages m ON m.sequence=t.source_sequence
 WHERE t.kind='message' AND t.sequence<=? AND m.role='assistant'
 AND COALESCE(json_extract(m.body,'$.incomplete'),0)=0
 ORDER BY t.sequence DESC LIMIT 1`, through).Scan(&sequence)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return sequence, err
}

func (s *Store) decisionPage(ctx context.Context, completed, bound int64, tail bool) ([]storedRecord, error) {
	const selectRecords = `SELECT t.sequence,t.kind,
 CASE WHEN t.sequence<? AND (` + browserObservationSQL + `) THEN
   json_object('role','tool','id',json_extract(m.body,'$.id'),
     'tool_call_id',json_extract(m.body,'$.tool_call_id'),'tool_name','` + toolapi.BrowserInspect + `','content',?)
 ELSE m.body END,
 m.refs,m.raw_bytes,` + browserObservationSQL + `
 FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence`
	order := " WHERE t.sequence>? ORDER BY t.sequence LIMIT 64"
	if tail {
		order = " WHERE t.sequence<? ORDER BY t.sequence DESC LIMIT 64"
	}
	page, err := s.readRecordPage(ctx, true, selectRecords+order, completed, toolapi.BrowserObservationRead, bound)
	for i := range page {
		record := &page[i]
		if record.browserObservation && record.sequence < completed {
			record.browserObservation = false
			record.refs = []byte("{}")
			record.size = len(record.body)
		}
	}
	return page, err
}
