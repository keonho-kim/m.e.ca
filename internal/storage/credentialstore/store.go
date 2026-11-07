// Purpose: Persist provider credentials as authenticated ciphertext with independent OS-held keys.
// Pattern: Encrypted credential repository over SQLite and a consumer-owned key boundary.
// Usage: config/credentials opens one Store per credential operation or maintenance pass and closes it.
// Related: internal/storage/credentialstore/database.go, internal/storage/credentialstore/crypto.go.
package credentialstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/storage/capacity"
	"strings"
	"unicode/utf8"
)

type Keys interface {
	Set(context.Context, string, string) error
	Get(context.Context, string) (string, error)
}

type Store struct {
	db        *sql.DB
	keys      Keys
	directory string
}

var (
	ErrKeyUnavailable = errors.New("OS encryption key unavailable; preserve the credential database and restore the OS credential store")
	ErrIntegrity      = errors.New("credential authentication failed; existing data was preserved")
)

func (s *Store) Close() error { return s.db.Close() }

func checkRef(ref string) error {
	if ref == "" || len(ref) > 128 || !utf8.ValidString(ref) || strings.ContainsAny(ref, "\r\n\x00") {
		return fmt.Errorf("invalid credential reference")
	}
	return nil
}

func (s *Store) Put(ctx context.Context, ref, scope, secret string) error {
	if err := checkIdentity(ref, scope); err != nil {
		return err
	}
	if secret == "" || len(secret) > 2048 || !utf8.ValidString(secret) || strings.ContainsAny(secret, "\r\n\x00") {
		return fmt.Errorf("invalid credential")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.write(ctx, func() error {
		if err := s.checkWrite(ctx, ref, scope); err != nil {
			return err
		}
		return s.put(ctx, ref, scope, secret)
	})
}

func (s *Store) put(ctx context.Context, ref, scope, secret string) error {
	keyRef, sealed, err := s.seal(ctx, ref, scope, secret)
	if err != nil {
		return err
	}
	// Only ciphertext reaches SQLite, its journal and bound SQL parameters.
	// Each update has a new OS key. Never delete either key on an uncertain
	// commit: the previous row and an older database backup must stay readable.
	result, err := s.db.ExecContext(ctx, `INSERT INTO credentials(ref,scope,key_ref,sealed) SELECT ?,?,?,?
WHERE EXISTS(SELECT 1 FROM credentials WHERE ref=?) OR (SELECT COUNT(*) FROM credentials)<?
ON CONFLICT(ref) DO UPDATE SET key_ref=excluded.key_ref,sealed=excluded.sealed
WHERE credentials.scope=excluded.scope`, ref, scope, keyRef, sealed, ref, maxCredentials)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		var present int
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM credentials WHERE ref=?", ref).Scan(&present); err != nil {
			return err
		}
		if present == 0 {
			return capacity.ErrLimit
		}
		return ErrIntegrity
	}
	return nil
}

func (s *Store) Get(ctx context.Context, ref, scope string) (string, error) {
	if err := checkIdentity(ref, scope); err != nil {
		return "", err
	}
	var keyRef, storedScope string
	var sealed []byte
	if err := s.db.QueryRowContext(ctx, "SELECT key_ref,scope,CASE WHEN length(sealed) BETWEEN 29 AND 2076 THEN sealed END FROM credentials WHERE ref=?", ref).Scan(&keyRef, &storedScope, &sealed); err != nil {
		return "", err
	}
	if storedScope != scope {
		return "", ErrIntegrity
	}
	return s.open(ctx, ref, scope, keyRef, sealed)
}

func (s *Store) Delete(ctx context.Context, ref, scope string) error {
	if err := checkIdentity(ref, scope); err != nil {
		return err
	}
	// Retired OS keys also protect explicit old backups; only uninstall clears them.
	return s.write(ctx, func() error {
		_, err := s.db.ExecContext(ctx, "DELETE FROM credentials WHERE ref=? AND scope=?", ref, scope)
		return err
	})
}

func checkIdentity(ref, scope string) error {
	if err := checkRef(ref); err != nil {
		return err
	}
	return checkRef(scope)
}
