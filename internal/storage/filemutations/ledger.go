// Purpose: Admit file operations exactly once and retain compact terminal receipts for lookup.
// Pattern: Transactional idempotency repository with explicit capacity.
// Usage: server.MutateEntry commits intent before filesystem work and saves the observed result after it.
// Related: internal/storage/filemutations/database.go, internal/commands/server/file_mutation_receipt.go.
package filemutations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/commands/server"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func identity(session, key string) [32]byte { return sha256.Sum256([]byte(session + "\x00" + key)) }
func decode(body []byte, key string) (server.FileMutationResult, error) {
	var result server.FileMutationResult
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if len(body) > 16384 || decoder.Decode(&result) != nil {
		return result, fault.New(diag.IntegrityFailed, diag.FileMutationStorage)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return result, fault.New(diag.IntegrityFailed, diag.FileMutationStorage)
	}
	switch result.State {
	case "pending", "applied", "rejected", "partial", "uncertain":
	default:
		return result, fault.New(diag.IntegrityFailed, diag.FileMutationStorage)
	}
	if len(result.Path) > 4096 || len(result.Source) > 4096 || !utf8.ValidString(result.Path) || !utf8.ValidString(result.Source) || strings.IndexByte(result.Path, 0) >= 0 || strings.IndexByte(result.Source, 0) >= 0 {
		return result, fault.New(diag.IntegrityFailed, diag.FileMutationStorage)
	}
	if result.Failure != nil && (!diag.ValidCode(result.Failure.Code) || result.Failure.Code == "" || len(result.Failure.Detail) > 8192 || len(result.Failure.Next) > 8192) {
		return result, fault.New(diag.IntegrityFailed, diag.FileMutationStorage)
	}
	result.Key = key
	return result, nil
}
func (l *Ledger) Admit(ctx context.Context, session, key string, digest [32]byte) (value server.FileMutationResult, duplicate bool, resultErr error) {
	id := identity(session, key)
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return value, false, err
	}
	defer tx.Rollback()
	var saved, body []byte
	err = tx.QueryRowContext(ctx, "SELECT digest,body FROM receipts WHERE id=?", id[:]).Scan(&saved, &body)
	if err == nil {
		if !bytes.Equal(saved, digest[:]) {
			return value, false, fault.New(diag.Conflict, diag.FileMutationConflict)
		}
		value, err = decode(body, key)
		return value, true, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return value, false, err
	}
	var count, used int
	if err := tx.QueryRowContext(ctx, "SELECT entries,bytes FROM budget WHERE id=1").Scan(&count, &used); err != nil {
		return value, false, err
	}
	if count >= 100000 || used+16384 > 32<<20 {
		return value, false, fault.New(diag.ResourceExhausted, diag.FileMutationCapacity)
	}
	value = server.FileMutationResult{Key: key, State: "pending"}
	stored := value
	stored.Key = ""
	body, err = json.Marshal(stored)
	if err != nil {
		return value, false, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO receipts(id,digest,charge,body) VALUES(?,?,16384,?)", id[:], digest[:], body); err != nil {
		return value, false, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE budget SET entries=entries+1,bytes=bytes+16384 WHERE id=1"); err != nil {
		return value, false, err
	}
	return value, false, tx.Commit()
}
func (l *Ledger) Lookup(ctx context.Context, session, key string) (server.FileMutationResult, error) {
	id := identity(session, key)
	var body []byte
	err := l.db.QueryRowContext(ctx, "SELECT body FROM receipts WHERE id=?", id[:]).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return server.FileMutationResult{}, fault.New(diag.NotFound, diag.ServerPageInvalid)
	}
	if err != nil {
		return server.FileMutationResult{}, err
	}
	return decode(body, key)
}
func (l *Ledger) Finish(ctx context.Context, session, key string, value server.FileMutationResult) error {
	value.Key = ""
	value.Paths = nil
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(body) > 16384 {
		return fault.New(diag.ResourceExhausted, diag.FileMutationCapacity)
	}
	id := identity(session, key)
	if _, err := decode(body, key); err != nil {
		return err
	}
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var oldCharge int
	if err := tx.QueryRowContext(ctx, "SELECT charge FROM receipts WHERE id=?", id[:]).Scan(&oldCharge); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "UPDATE receipts SET body=?,charge=? WHERE id=?", body, len(body), id[:])
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil {
		return err
	} else if count != 1 {
		return fault.New(diag.NotFound, diag.ServerPageInvalid)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE budget SET bytes=bytes+? WHERE id=1", len(body)-oldCharge); err != nil {
		return err
	}
	return tx.Commit()
}
