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

	"reflect"

	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (l *Ledger) ReadResponse(ctx context.Context, id string) (server.Response, error) {
	raw, err := identity(id, "resp_")
	if err != nil {
		return server.Response{}, err
	}
	v, err := read(ctx, l.db, raw)
	if errors.Is(err, sql.ErrNoRows) {
		return server.Response{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerResponseWasNotFound)
	}
	if err != nil {
		return server.Response{}, err
	}
	return decodeResponse(v, id, l.scope)
}
func (l *Ledger) ReserveResponse(ctx context.Context, value server.Response) (result server.Response, duplicate bool, resultErr error) {
	if err := checkResponse(value); err != nil {
		return server.Response{}, false, err
	}
	if value.Scope != l.scope {
		return server.Response{}, false, faults.Errorf(diagcatalog.ServerrequestsResponseBelongsToAnotherProject)
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
			return old, true, faults.New(diagcatalog.Conflict, diagcatalog.ServerIdempotencyKeyWasAlreadyUsedWithA)
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
			return server.Response{}, false, faults.New(diagcatalog.Conflict, diagcatalog.ServerrequestsPreviousResponseIsUnavailable)
		}
		parent, err := decodeResponse(v, value.Previous, l.scope)
		if err != nil {
			return server.Response{}, false, err
		}
		if parent.Kind != "response" || !parent.Stored || parent.Status != "completed" || parent.Next != "" {
			return server.Response{}, false, faults.New(diagcatalog.Conflict, diagcatalog.ServerPreviousResponseIdMustNameTheLatest)
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
	reservation, err := l.reserve(ctx)
	if err != nil {
		return server.Response{}, false, err
	}
	defer func() { resultErr = errors.Join(resultErr, reservation.Close()) }()
	_, err = tx.ExecContext(ctx, "INSERT INTO requests(id,kind,digest,response) VALUES(?,?,?,?)", raw, value.Kind, digest, body)
	if err != nil {
		return server.Response{}, false, err
	}
	reservation.Publish()
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
			return faults.Errorf(diagcatalog.ServerrequestsTerminalResponseCannotChangeOriginalPreserved)
		}
		return nil
	}
	if old.Scope != value.Scope || old.Digest != value.Digest || old.Kind != value.Kind || old.Stored != value.Stored || old.Background != value.Background || old.Previous != value.Previous || old.Created != value.Created || old.Session != "" && old.Session != value.Session || old.Task != "" && old.Task != value.Task || old.Next != "" && old.Next != value.Next {
		return faults.Errorf(diagcatalog.ServerrequestsResponseIdentityChangedOriginalPreserved)
	}
	if err := writeResponse(ctx, tx, value); err != nil {
		return err
	}
	if value.Status == "failed" && old.Status == "queued" && old.Task == "" && value.Task == "" && value.Previous != "" {
		// This admission never acquired an execution identity. Release only its
		// own head claim, in the same transaction as the durable failure.
		parentRaw, _ := identity(value.Previous, "resp_")
		stored, err := read(ctx, tx, parentRaw)
		if err != nil {
			return err
		}
		parent, err := decodeResponse(stored, value.Previous, l.scope)
		if err != nil {
			return err
		}
		if parent.Next == value.ID {
			parent.Next = ""
			if err := writeResponse(ctx, tx, parent); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
