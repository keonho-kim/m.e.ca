// Purpose: Resume a time-bounded maintenance pass beyond a slow or unavailable credential.
// Pattern: Private checkpoint under the existing maintenance lease.
// Usage: RotateDue records its last attempted row only when its pass deadline expires.
// Related: internal/storage/credentialstore/rotation.go, internal/storage/credentialstore/rotation_schedule.go.
package credentialstore

import (
	"context"
	"errors"

	"io"
	"os"
	"strings"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func rotationCursor(root *os.Root) (cursor string, resultErr error) {
	info, err := root.Lstat("cursor")
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 256 {
		return "", faults.Errorf(diagcatalog.CredentialstoreInvalidPrivateRotationCursorContentsPreserved)
	}
	file, err := root.Open("cursor")
	if err != nil {
		return "", err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	current, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !os.SameFile(info, current) {
		return "", faults.Errorf(diagcatalog.CredentialstoreRotationCursorChangedWhileOpening)
	}
	body, err := io.ReadAll(io.LimitReader(file, 257))
	if err != nil {
		return "", err
	}
	parts := strings.Split(string(body), "\n")
	if len(body) > 256 || len(parts) != 3 || parts[0] != "v1" || parts[2] != "" {
		return "", faults.Errorf(diagcatalog.CredentialstoreUnsupportedRotationCursorContentsPreserved)
	}
	if parts[1] != "" {
		if err := checkRef(parts[1]); err != nil {
			return "", err
		}
	}
	return parts[1], nil
}

func checkpointRotation(ctx context.Context, directory, after string, cause error) error {
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return cause
	}
	return errors.Join(cause, writeRotationCursor(directory, after))
}

func writeRotationCursor(directory, cursor string) error {
	return statefs.Write(directory, "cursor", []byte("v1\n"+cursor+"\n"))
}
