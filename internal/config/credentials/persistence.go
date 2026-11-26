// Purpose: Compose encrypted provider-key persistence with private local encryption keys.
// Pattern: Credential Adapter.
// Usage: Connection setup and model selection call Vault.Get and Set.
// Related: internal/config/credentials/vault.go, internal/provider/setup/setup.go, internal/commands/install/uninstall.go.
package credentials

import (
	"context"
	"database/sql"
	"errors"

	"os"
	"path/filepath"
	"strings"
	"time"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/credentialkeys"
	"github.com/keonho-kim/meca/internal/storage/credentialstore"
)

func (v *Vault) keyStore() credentialstore.Keys {
	if v.Keys != nil {
		return v.Keys
	}
	return credentialkeys.Store{Directory: filepath.Join(v.State, credentialkeys.DirectoryName)}
}

func (v *Vault) Set(ctx context.Context, ref, scope, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if ref == "" || len(key) > 2048 || strings.ContainsAny(key, "\r\n\x00") {
		return faults.Errorf(diagcatalog.ServerInvalidCredential)
	}
	key = strings.Trim(key, " \t")
	if err := v.Remember(key); err != nil {
		return err
	}
	if v.State == "" {
		return faults.Errorf(diagcatalog.CredentialsCredentialStateDirectoryIsRequired)
	}
	store, err := credentialstore.Open(ctx, v.State, v.keyStore(), key != "")
	if key == "" && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if key == "" {
		return errors.Join(store.Delete(ctx, ref, scope), store.Close())
	}
	return errors.Join(store.Put(ctx, ref, scope, key), store.Close())
}

func (v *Vault) Get(ctx context.Context, ref, scope string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if ref == "" {
		return "", nil
	}
	if v.State == "" {
		return "", faults.Errorf(diagcatalog.CredentialsCredentialStateDirectoryIsRequired)
	}
	store, err := credentialstore.Open(ctx, v.State, v.keyStore(), false)
	if err != nil {
		return "", faults.Errorf(diagcatalog.CredentialsEncryptedCredentialUnavailablePreserveExistingStateAnd, err)
	}
	key, err := store.Get(ctx, ref, scope)
	if err = errors.Join(err, store.Close()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", faults.Errorf(diagcatalog.CredentialsEncryptedCredentialRecordWasNotFoundExisting)
		}
		return "", err
	}
	if err := v.Remember(key); err != nil {
		return "", err
	}
	return strings.Trim(key, " \t"), nil
}
