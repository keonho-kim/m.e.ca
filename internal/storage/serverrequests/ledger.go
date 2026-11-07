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
	"fmt"
	"github.com/keonho-kim/meca/internal/commands/server"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"strings"
)

type rowQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type stored struct {
	kind, session, status, message string
	digest, response               []byte
	version                        int
}

func read(ctx context.Context, q rowQuery, id []byte) (stored, error) {
	var v stored
	err := q.QueryRowContext(ctx, `SELECT
 CASE WHEN length(CAST(kind AS BLOB))<=8 THEN kind END,
 CASE WHEN length(digest)=32 THEN digest END,
 CASE WHEN length(CAST(session AS BLOB))<=34 THEN session END,
 CASE WHEN length(CAST(status AS BLOB))<=16 THEN status END,
 CASE WHEN length(CAST(error AS BLOB))<=4096 THEN error END,
 CASE WHEN length(response)<=? THEN response END,version
 FROM requests WHERE id=?`, maxResponseRecordBytes, id).Scan(&v.kind, &v.digest, &v.session, &v.status, &v.message, &v.response, &v.version)
	if err == nil && (v.version != 1 || len(v.digest) != 32 || v.kind != "native" && v.response == nil) {
		err = fmt.Errorf("unsupported request record; original preserved")
	}
	return v, err
}
func identity(id, prefix string) ([]byte, error) {
	if len(id) != 69 || !strings.HasPrefix(id, prefix) {
		return nil, &server.Error{Code: "invalid_argument", Message: "invalid request identity"}
	}
	raw, err := hex.DecodeString(id[5:])
	if err != nil {
		return nil, &server.Error{Code: "invalid_argument", Message: "invalid request digest"}
	}
	return raw, nil
}
func restored(t server.Task) server.Task {
	if t.Status == "accepted" {
		t.Status = "interrupted"
	}
	return t
}
func (l *Ledger) Admit(ctx context.Context, session, key, text string) (result server.Task, duplicate bool, resultErr error) {
	if !sessionstate.ValidID(session) {
		return result, false, fmt.Errorf("invalid request session")
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
			return result, true, &server.Error{Code: "conflict", Message: "request key was already used with different text"}
		}
		return restored(server.Task{ID: id, Session: session, Status: old.status, Error: old.message}), true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, false, err
	}
	if err := l.reserve(ctx); err != nil {
		return result, false, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO requests(id,kind,digest,session,status) VALUES(?,'native',?,?,'accepted')", raw, digest[:], session)
	if err != nil {
		return result, false, err
	}
	if err := tx.Commit(); err != nil {
		return result, false, err
	}
	return server.Task{ID: id, Session: session, Status: "accepted"}, false, nil
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
		if status == "" {
			status = "interrupted"
		}
		return server.Task{ID: id, Session: value.Session, Status: status, Error: value.Error}, nil
	}
	if !sessionstate.ValidID(v.session) {
		return server.Task{}, fmt.Errorf("invalid stored session")
	}
	switch v.status {
	case "accepted", "answered", "stopped", "interrupted":
	default:
		return server.Task{}, fmt.Errorf("invalid stored task status")
	}
	return restored(server.Task{ID: id, Session: v.session, Status: v.status, Error: v.message}), nil
}
func (l *Ledger) Finish(ctx context.Context, task server.Task) error {
	raw, err := identity(task.ID, "task_")
	if err != nil {
		return err
	}
	switch task.Status {
	case "answered", "stopped", "interrupted":
	default:
		return fmt.Errorf("invalid completion status")
	}
	if len(task.Error) > 4096 {
		return fmt.Errorf("request completion exceeds limit")
	}
	result, err := l.db.ExecContext(ctx, "UPDATE requests SET status=?,error=? WHERE id=? AND kind='native' AND session=? AND (status='accepted' OR (status=? AND error=?))", task.Status, task.Error, raw, task.Session, task.Status, task.Error)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n != 1 {
		return server.ErrTaskNotFound
	}
	return err
}
