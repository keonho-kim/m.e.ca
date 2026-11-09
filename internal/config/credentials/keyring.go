// Purpose: Compose encrypted provider-key persistence with MECA-owned OS encryption keys.
// Pattern: Credential Adapter.
// Usage: Connection setup and model selection call Vault.Get and Set.
// Related: internal/config/credentials/vault.go, internal/provider/setup/setup.go, internal/commands/install/uninstall.go.
package credentials

import (
	"context"
	"database/sql"
	"errors"

	"os"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/config/credentials/osstore"
	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/credentialstore"
)

const service = "meca"

type KeyStore interface {
	Set(context.Context, string, string) error
	Get(context.Context, string) (string, error)
	Clear(context.Context) error
}

func (v *Vault) keyStore() KeyStore {
	if v.Keys != nil {
		return v.Keys
	}
	return osstore.Store{Service: service}
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
	if err := statefs.Write(v.State, "credentials.used", []byte(service+"\n")); err != nil {
		return err
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

// Clear removes only credentials belonging to MECA's fixed keyring service.
func (v *Vault) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := v.keyStore().Clear(ctx)
	if errors.Is(err, osstore.ErrNotFound) {
		return nil
	}
	return err
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
