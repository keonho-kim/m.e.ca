// Purpose: Choose an archived prefix while retaining recent whole messages verbatim.
// Pattern: Repository projection.
// Usage: Compaction asks for a byte-budgeted recent tail before generating summaries.
// Related: internal/agent/compact/controller.go, internal/storage/sqlite/context.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/storage/history"
)

func (s *Store) CompactionBoundary(ctx context.Context, session string, through int64, keepBytes int) (boundary int64, err error) {
	if through < 0 || keepBytes < 0 {
		return 0, fmt.Errorf("invalid compaction tail bounds")
	}
	if keepBytes == 0 {
		return through, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT t.sequence,m.body FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence<=? ORDER BY t.sequence DESC`, session, through)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	selection := history.TailSelection{Target: keepBytes}
	for rows.Next() {
		var sequence int64
		var body []byte
		if err := rows.Scan(&sequence, &body); err != nil {
			return 0, err
		}
		var message modelapi.Message
		if err := json.Unmarshal(body, &message); err != nil {
			return 0, err
		}
		if boundary, ready := selection.Keep(sequence, len(body), message); ready {
			return boundary, nil
		}
	}
	return 0, rows.Err()
}
