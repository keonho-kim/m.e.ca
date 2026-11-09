// Purpose: Durably admit native requests and expose native task identities for both request kinds.
// Pattern: Project-bound SQLite repository with immutable admission identities.
// Usage: Server commands persist admission before starting any model or tool work.
// Related: internal/storage/serverrequests/database.go, internal/storage/serverrequests/responses.go.
package serverrequests

import (
	"context"
	"crypto/sha256"
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
	kind, session, status, message, code string
	digest, response                     []byte
	version                              int
}

func read(ctx context.Context, q rowQuery, id []byte) (stored, error) {
	var v stored
	err := q.QueryRowContext(ctx, `SELECT
 CASE WHEN length(CAST(kind AS BLOB))<=8 THEN kind END,
 CASE WHEN length(digest)=32 THEN digest END,
 CASE WHEN length(CAST(session AS BLOB))<=34 THEN session END,
 CASE WHEN length(CAST(status AS BLOB))<=16 THEN status END,
 CASE WHEN length(CAST(error AS BLOB))<=4096 THEN error END,
 CASE WHEN length(CAST(error_code AS BLOB))<=128 THEN error_code END,
 CASE WHEN length(response)<=? THEN response END,version
 FROM requests WHERE id=?`, maxResponseRecordBytes, id).Scan(&v.kind, &v.digest, &v.session, &v.status, &v.message, &v.code, &v.response, &v.version)
	if err == nil && (v.version != 1 || len(v.digest) != 32 || !diagcatalog.ValidCode(v.code) || v.kind != "native" && v.response == nil) {
		err = faults.Errorf(diagcatalog.ServerrequestsUnsupportedRequestRecordOriginalPreserved)
	}
	return v, err
}
func identity(id, prefix string) ([]byte, error) {
	if len(id) != 69 || !strings.HasPrefix(id, prefix) {
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
func (l *Ledger) Admit(ctx context.Context, session, key, text string) (result server.Task, duplicate bool, resultErr error) {
	if !sessionstate.ValidID(session) {
		return result, false, faults.Errorf(diagcatalog.ServerrequestsInvalidRequestSession)
	}
	id := server.TaskID(session, key)
	raw, _ := identity(id, "task_")
	digest := sha256.Sum256([]byte(text))
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return result, false, err
	}
	defer tx.Rollback()
	old, err := read(ctx, tx, raw)
	if err == nil {
		if old.kind != "native" || old.session != session || string(old.digest) != string(digest[:]) {
			return result, true, faults.New(diagcatalog.Conflict, diagcatalog.ServerrequestsRequestKeyWasAlreadyUsedWithDifferent)
		}
		return restored(server.Task{ID: id, Session: session, Status: old.status, Error: old.message, Code: old.code}), true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, false, err
	}
	reservation, err := l.reserve(ctx)
	if err != nil {
		return result, false, err
	}
	defer func() { resultErr = errors.Join(resultErr, reservation.Close()) }()
	_, err = tx.ExecContext(ctx, "INSERT INTO requests(id,kind,digest,session,status) VALUES(?,'native',?,?,'accepted')", raw, digest[:], session)
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
	raw, err := identity(id, "task_")
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
	if v.kind != "native" {
		value, err := decodeResponse(v, "resp_"+id[5:], l.scope)
		if err != nil {
			return server.Task{}, err
		}
		if value.Task == "" {
			return server.Task{}, server.ErrTaskNotFound
		}
		status := value.TaskStatus
		if status == "" || status == "accepted" {
			status = diagcatalog.Interrupted
			value.Code = diagcatalog.Interrupted
			return server.Task{ID: id, Session: value.Session, Status: status, Error: value.Error, Code: value.Code, Persistence: server.PersistenceUncertain}, nil
		}
		return server.Task{ID: id, Session: value.Session, Status: status, Error: value.Error, Code: value.Code, Persistence: server.PersistenceConfirmed}, nil
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
func (l *Ledger) Finish(ctx context.Context, task server.Task) error {
	raw, err := identity(task.ID, "task_")
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
	result, err := l.db.ExecContext(ctx, "UPDATE requests SET status=?,error=?,error_code=? WHERE id=? AND kind='native' AND session=? AND (status='accepted' OR (status=? AND error=? AND error_code=?))", task.Status, task.Error, task.Code, raw, task.Session, task.Status, task.Error, task.Code)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n != 1 {
		return server.ErrTaskNotFound
	}
	return err
}
