// Purpose: Persist native task admission, deduplication and completion without storing user input.
// Pattern: Project-bound transactional SQLite repository.
// Usage: Server Start commits one input digest before running model or tool work.
// Related: internal/storage/serverrequests/database.go, internal/commands/server/identity.go.
package serverrequests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type rowQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type stored struct {
	session, status, message, code string
	digest                         []byte
}

const taskColumns = `CASE WHEN length(digest)=32 THEN digest END,
 CASE WHEN length(CAST(session AS BLOB))<=34 THEN session END,
 CASE WHEN length(CAST(status AS BLOB))<=16 THEN status END,
 CASE WHEN length(CAST(error AS BLOB))<=4096 THEN error END,
 CASE WHEN length(CAST(error_code AS BLOB))<=128 THEN error_code END`

func read(ctx context.Context, q rowQuery, id []byte) (stored, error) {
	var value stored
	err := q.QueryRowContext(ctx, "SELECT "+taskColumns+" FROM requests WHERE id=?", id).Scan(&value.digest, &value.session, &value.status, &value.message, &value.code)
	return value, err
}

func taskRecord(id string, v stored) (server.Task, error) {
	if len(v.digest) != 32 || !diagcatalog.ValidCode(v.code) {
		return server.Task{}, faults.Errorf(diagcatalog.ServerrequestsUnsupportedRequestRecordOriginalPreserved)
	}
	if !sessionstate.ValidID(v.session) {
		return server.Task{}, faults.Errorf(diagcatalog.ServerrequestsInvalidStoredSession)
	}
	switch v.status {
	case "accepted", "answered", "stopped", diagcatalog.Interrupted:
	default:
		return server.Task{}, faults.Errorf(diagcatalog.ServerrequestsInvalidStoredTaskStatus)
	}
	return restored(server.Task{ID: id, Session: v.session, Status: v.status, Error: v.message, Code: v.code}), nil
}

func identity(id string) ([]byte, error) {
	if len(id) != 69 || !strings.HasPrefix(id, "task_") {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerrequestsInvalidRequestIdentity)
	}
	raw, err := hex.DecodeString(id[5:])
	if err != nil {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerrequestsInvalidRequestDigest)
	}
	return raw, nil
}
func restored(t server.Task) server.Task {
	t.Persistence = server.PersistenceConfirmed
	if t.Status == "accepted" {
		t.Status = diagcatalog.Interrupted
		t.Code = diagcatalog.Interrupted
		t.Persistence = server.PersistenceUncertain
	}
	return t
}
func (l *Ledger) Admit(ctx context.Context, session, key string, digest [32]byte) (result server.Task, duplicate bool, resultErr error) {
	if !sessionstate.ValidID(session) {
		return result, false, faults.Errorf(diagcatalog.ServerrequestsInvalidRequestSession)
	}
	id := server.TaskID(session, key)
	raw, _ := identity(id)
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return result, false, err
	}
	defer tx.Rollback()
	old, err := read(ctx, tx, raw)
	if err == nil {
		original, err := taskRecord(id, old)
		if err != nil {
			return result, true, err
		}
		if old.session != session || !bytes.Equal(old.digest, digest[:]) {
			return result, true, faults.New(diagcatalog.Conflict, diagcatalog.ServerrequestsRequestKeyWasAlreadyUsedWithDifferent)
		}
		return original, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, false, err
	}
	reservation, err := l.reserve(ctx)
	if err != nil {
		return result, false, err
	}
	defer func() { resultErr = errors.Join(resultErr, reservation.Close()) }()
	_, err = tx.ExecContext(ctx, "INSERT INTO requests(id,digest,session,status) VALUES(?,?,?,'accepted')", raw, digest[:], session)
	if err != nil {
		return result, false, err
	}
	reservation.Publish()
	if err := tx.Commit(); err != nil {
		return result, false, err
	}
	return server.Task{ID: id, Session: session, Status: "accepted", Persistence: server.PersistencePending}, false, nil
}
func (l *Ledger) Get(ctx context.Context, id string) (server.Task, error) {
	raw, err := identity(id)
	if err != nil {
		return server.Task{}, err
	}
	v, err := read(ctx, l.db, raw)
	if errors.Is(err, sql.ErrNoRows) {
		return server.Task{}, server.ErrTaskNotFound
	}
	if err != nil {
		return server.Task{}, err
	}
	return taskRecord(id, v)
}
func (l *Ledger) Finish(ctx context.Context, task server.Task) error {
	raw, err := identity(task.ID)
	if err != nil {
		return err
	}
	switch task.Status {
	case "answered", "stopped", diagcatalog.Interrupted:
	default:
		return faults.Errorf(diagcatalog.ServerrequestsInvalidCompletionStatus)
	}
	if len(task.Error) > 4096 || !diagcatalog.ValidCode(task.Code) {
		return faults.Errorf(diagcatalog.ServerrequestsRequestCompletionExceedsLimit)
	}
	result, err := l.db.ExecContext(ctx, "UPDATE requests SET status=?,error=?,error_code=? WHERE id=? AND session=? AND (status='accepted' OR (status=? AND error=? AND error_code=?))", task.Status, task.Error, task.Code, raw, task.Session, task.Status, task.Error, task.Code)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n != 1 {
		return server.ErrTaskNotFound
	}
	return err
}
