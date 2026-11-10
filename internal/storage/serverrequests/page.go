// Purpose: Page durable task identities without loading journals or changing deduplication records.
// Pattern: Repository read projection.
// Usage: Native task history and server capacity discovery call Page and Usage.
// Related: internal/storage/serverrequests/ledger.go, internal/storage/serverrequests/capacity.go.
package serverrequests

import (
	"context"
	"encoding/hex"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/capacity"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (l *Ledger) Page(ctx context.Context, session, cursor string, limit int) ([]server.Task, string, error) {
	if limit < 1 || limit > 100 || session != "" && !sessionstate.ValidID(session) {
		return nil, "", faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	after := []byte{}
	if cursor != "" {
		var err error
		after, err = identity(cursor)
		if err != nil {
			return nil, "", err
		}
	}
	query := "SELECT id," + taskColumns + " FROM requests WHERE id>? ORDER BY id LIMIT ?"
	args := []any{after, limit + 1}
	if session != "" {
		query = "SELECT id," + taskColumns + " FROM requests WHERE session=? AND id>? ORDER BY id LIMIT ?"
		args = []any{session, after, limit + 1}
	}
	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	var values []server.Task
	for rows.Next() {
		var id []byte
		var record stored
		if err := rows.Scan(&id, &record.digest, &record.session, &record.status, &record.message, &record.code); err != nil {
			_ = rows.Close()
			return nil, "", err
		}
		if len(id) != 32 {
			_ = rows.Close()
			return nil, "", faults.Errorf(diagcatalog.ServerrequestsUnsupportedRequestRecordOriginalPreserved)
		}
		value, err := taskRecord("task_"+hex.EncodeToString(id), record)
		if err != nil {
			_ = rows.Close()
			return nil, "", err
		}
		values = append(values, value)
	}
	readErr := rows.Err()
	closeErr := rows.Close()
	if readErr != nil {
		return nil, "", readErr
	}
	if closeErr != nil {
		return nil, "", closeErr
	}
	next := ""
	if len(values) > limit {
		values = values[:limit]
		next = values[limit-1].ID
	}
	return values, next, nil
}

func (l *Ledger) Usage(ctx context.Context) (int64, int64, error) {
	used, err := capacity.Allocated(ctx, filepath.Join(l.directory, "capacity"))
	return used, maxAdmissions, err
}
