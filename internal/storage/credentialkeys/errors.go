// Purpose: Preserve actionable key-store failures without exposing raw storage diagnostics publicly.
// Pattern: Adapter error translation with retained causes.
// Usage: Store.Set and Get classify errors after closing their owned resources.
// Related: internal/storage/credentialkeys/store.go, internal/diagnostics/catalog/local_credentials.go.
package credentialkeys

import (
	"context"
	"errors"
	"os"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func storageError(err error, write bool) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var known *fault.Error
	if errors.As(err, &known) {
		return err
	}
	if errors.Is(err, os.ErrNotExist) {
		return fault.New(diag.NotFound, diag.LocalKeysMissing, err)
	}
	if errors.Is(err, os.ErrPermission) {
		return fault.New(diag.PermissionDenied, diag.LocalKeysAccessDenied, err)
	}
	code := fault.Code(err)
	if code == diag.Internal {
		code = diag.Unavailable
	}
	if write {
		return fault.New(code, diag.LocalKeysWriteFailed, err)
	}
	return fault.New(code, diag.LocalKeysReadFailed, err)
}
