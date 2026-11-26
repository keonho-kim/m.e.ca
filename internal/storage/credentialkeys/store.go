// Purpose: Store immutable encryption-key versions compactly in a private local database.
// Pattern: SQLite key-material adapter behind the credential repository's narrow contract.
// Usage: Vault and installation verification call Store.Set and Get.
// Related: internal/storage/credentialkeys/database.go, internal/config/credentials/persistence.go.
package credentialkeys

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"strings"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const DirectoryName = "credential-keys"
const Filename = "keys.db"

type Store struct{ Directory string }

func (s Store) Set(ctx context.Context, ref, value string) (resultErr error) {
	defer func() { resultErr = storageError(resultErr, true) }()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validReference(ref) || len(value) != 44 {
		return fault.Errorf(diag.LocalKeysInvalidKey)
	}
	material, err := base64.StdEncoding.Strict().DecodeString(value)
	defer clear(material)
	if err != nil || len(material) != 32 {
		return fault.Errorf(diag.LocalKeysInvalidKey)
	}
	db, err := s.open(ctx, true)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, db.Close()) }()
	// Existing versions remain immutable so older ciphertext backups stay usable.
	if _, err := db.ExecContext(ctx, "INSERT INTO keys(ref, material) VALUES (?, ?) ON CONFLICT(ref) DO NOTHING", ref, material); err != nil {
		return err
	}
	actual, err := read(ctx, db, ref)
	defer clear(actual)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(material, actual) != 1 {
		return fault.Errorf(diag.LocalKeysConflict)
	}
	return nil
}

func (s Store) Get(ctx context.Context, ref string) (value string, resultErr error) {
	defer func() { resultErr = storageError(resultErr, false) }()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !validReference(ref) {
		return "", fault.Errorf(diag.LocalKeysInvalidKey)
	}
	db, err := s.open(ctx, false)
	if err != nil {
		return "", err
	}
	defer func() { resultErr = errors.Join(resultErr, db.Close()) }()
	material, err := read(ctx, db, ref)
	defer clear(material)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(material), nil
}

func read(ctx context.Context, db *sql.DB, ref string) ([]byte, error) {
	var material []byte
	err := db.QueryRowContext(ctx, "SELECT CASE WHEN length(material)=32 THEN material END FROM keys WHERE ref=?", ref).Scan(&material)
	if errors.Is(err, sql.ErrNoRows) {
		// Keep this adapter's missing key distinct from a missing credential row.
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	if len(material) != 32 {
		clear(material)
		return nil, fault.Errorf(diag.LocalKeysInvalidKey)
	}
	return material, nil
}

func validReference(ref string) bool {
	if len(ref) != 37 || !strings.HasPrefix(ref, "encryption_") {
		return false
	}
	for _, character := range ref[len("encryption_"):] {
		if !(character >= 'A' && character <= 'Z' || character >= '2' && character <= '7') {
			return false
		}
	}
	return true
}
