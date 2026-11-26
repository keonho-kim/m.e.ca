// Purpose: Verify the runtime key store's durable write and read operations during installation.
// Pattern: Isolated storage probe using the production key adapter.
// Usage: Install holds the application lease while Prepare verifies local credentials.
// Related: internal/storage/credentialkeys/store.go, internal/commands/install/install.go.
package credentials

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/credentialkeys"
)

func Prepare(ctx context.Context, state string) (resultErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := statefs.Open(filepath.Join(state, credentialkeys.DirectoryName))
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	name := ".install-check-" + rand.Text()
	if err := root.Mkdir(name, 0700); err != nil {
		return err
	}
	defer func() {
		resultErr = errors.Join(resultErr, root.RemoveAll(name), statefs.SyncDirectory(root))
	}()
	store := credentialkeys.Store{Directory: filepath.Join(root.Name(), name)}
	ref := "encryption_" + rand.Text()
	var material [32]byte
	if _, err := rand.Read(material[:]); err != nil {
		return err
	}
	defer clear(material[:])
	value := base64.StdEncoding.EncodeToString(material[:])
	if err := store.Set(ctx, ref, value); err != nil {
		return err
	}
	actual, err := store.Get(ctx, ref)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(actual), []byte(value)) != 1 {
		return fault.Errorf(diag.LocalKeysVerificationFailed)
	}
	return ctx.Err()
}
