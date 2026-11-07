// Purpose: Compose encrypted provider-key persistence with MECA-owned OS encryption keys.
// Pattern: Credential Adapter.
// Usage: Connection setup and model selection call Vault.Get and Set.
// Related: internal/config/credentials/vault.go, internal/provider/setup/setup.go, internal/commands/install/uninstall.go.
package credentials

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/config/credentials/osstore"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/storage/credentialstore"
	"os"
	"strings"
	"time"
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
		return fmt.Errorf("invalid credential")
	}
	key = strings.Trim(key, " \t")
	if err := v.Remember(key); err != nil {
		return err
	}
	if v.State == "" {
		return fmt.Errorf("credential state directory is required")
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
		return "", fmt.Errorf("credential state directory is required")
	}
	store, err := credentialstore.Open(ctx, v.State, v.keyStore(), false)
	if err != nil {
		return "", fmt.Errorf("encrypted credential unavailable; preserve existing state and reconnect using !model: %w", err)
	}
	key, err := store.Get(ctx, ref, scope)
	if err = errors.Join(err, store.Close()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("encrypted credential record was not found; existing OS entries are preserved; reconnect using !model")
		}
		return "", err
	}
	if err := v.Remember(key); err != nil {
		return "", err
	}
	return strings.Trim(key, " \t"), nil
}
