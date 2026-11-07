// Purpose: Reserve compatible requests and their continuation heads atomically without persisting input text.
// Pattern: Transactional request metadata repository.
// Usage: Server admission and completion share one durable row with native task lookup.
// Related: internal/storage/serverrequests/validation.go, internal/commands/server/submission.go.
package serverrequests

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/commands/server"
	"reflect"
)

func (l *Ledger) ReadResponse(ctx context.Context, id string) (server.Response, error) {
	raw, err := identity(id, "resp_")
	if err != nil {
		return server.Response{}, err
	}
	v, err := read(ctx, l.db, raw)
	if errors.Is(err, sql.ErrNoRows) {
		return server.Response{}, &server.Error{Code: "not_found", Message: "response was not found"}
	}
	if err != nil {
		return server.Response{}, err
	}
	return decodeResponse(v, id, l.scope)
}
func (l *Ledger) ReserveResponse(ctx context.Context, value server.Response) (server.Response, bool, error) {
	if err := checkResponse(value); err != nil {
		return server.Response{}, false, err
	}
	if value.Scope != l.scope {
		return server.Response{}, false, fmt.Errorf("response belongs to another project")
	}
	raw, _ := identity(value.ID, "resp_")
	digest, _ := hex.DecodeString(value.Digest)
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return server.Response{}, false, err
	}
	defer tx.Rollback()
	v, err := read(ctx, tx, raw)
	if err == nil {
		old, err := decodeResponse(v, value.ID, l.scope)
		if err != nil {
			return old, true, err
		}
		if old.Digest != value.Digest || old.Kind != value.Kind {
			return old, true, &server.Error{Code: "conflict", Message: "Idempotency-Key was already used with a different request"}
		}
		return old, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return server.Response{}, false, err
	}
	if value.Previous != "" {
		parentRaw, _ := identity(value.Previous, "resp_")
		v, err := read(ctx, tx, parentRaw)
		if err != nil {
			return server.Response{}, false, &server.Error{Code: "conflict", Message: "previous response is unavailable"}
		}
		parent, err := decodeResponse(v, value.Previous, l.scope)
		if err != nil {
			return server.Response{}, false, err
		}
		if parent.Kind != "response" || !parent.Stored || parent.Status != "completed" || parent.Next != "" {
			return server.Response{}, false, &server.Error{Code: "conflict", Message: "previous_response_id must name the latest completed response"}
		}
		parent.Next = value.ID
		if err := writeResponse(ctx, tx, parent); err != nil {
			return server.Response{}, false, err
		}
	}
	body, err := encodeResponse(value)
	if err != nil {
		return server.Response{}, false, err
	}
	if err := l.reserve(ctx); err != nil {
		return server.Response{}, false, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO requests(id,kind,digest,response) VALUES(?,?,?,?)", raw, value.Kind, digest, body)
	if err != nil {
		return server.Response{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return server.Response{}, false, err
	}
	return value, false, nil
}
func writeResponse(ctx context.Context, tx *sql.Tx, value server.Response) error {
	raw, _ := identity(value.ID, "resp_")
	body, err := encodeResponse(value)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE requests SET response=? WHERE id=?", body, raw)
	return err
}
func (l *Ledger) SaveResponse(ctx context.Context, value server.Response) error {
	if err := checkResponse(value); err != nil {
		return err
	}
	raw, _ := identity(value.ID, "resp_")
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	v, err := read(ctx, tx, raw)
	if err != nil {
		return err
	}
	old, err := decodeResponse(v, value.ID, l.scope)
	if err != nil {
		return err
	}
	if old.Status != "queued" && old.Status != "in_progress" {
		if !reflect.DeepEqual(old, value) {
			return fmt.Errorf("terminal response cannot change; original preserved")
		}
		return nil
	}
	if old.Scope != value.Scope || old.Digest != value.Digest || old.Kind != value.Kind || old.Stored != value.Stored || old.Background != value.Background || old.Previous != value.Previous || old.Created != value.Created || old.Session != "" && old.Session != value.Session || old.Task != "" && old.Task != value.Task || old.Next != "" && old.Next != value.Next {
		return fmt.Errorf("response identity changed; original preserved")
	}
	if err := writeResponse(ctx, tx, value); err != nil {
		return err
	}
	return tx.Commit()
}
