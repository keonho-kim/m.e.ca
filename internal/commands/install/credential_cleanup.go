// Purpose: Delete OS keys only after private state removal is durable, and retain cleanup intent for retry.
// Pattern: Irreversible uninstall state transition with a durable ownership marker.
// Usage: Uninstall resumes pending cleanup; startup and Install reject an unfinished removal.
// Related: internal/commands/install/uninstall.go, internal/commands/install/install.go, cmd/meca/main.go.
package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"os"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const cleanupName = ".meca-credential-cleanup"

func pendingCleanup(root *os.Root) (string, bool, error) {
	info, err := root.Lstat(cleanupName)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() != 65 {
		return "", false, faults.Errorf(diagcatalog.InstallInvalidPrivateCredentialCleanupMarkerPreserveIt)
	}
	body, _, _, err := readFile(root, cleanupName)
	if err != nil {
		return "", false, err
	}
	if len(body) != 65 {
		return "", false, faults.Errorf(diagcatalog.InstallCredentialCleanupMarkerChangedWhileReading)
	}
	digest := string(body[:64])
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 || body[64] != '\n' {
		return "", false, faults.Errorf(diagcatalog.InstallInvalidCredentialCleanupIdentity)
	}
	return digest, true, nil
}

func syncDirectory(root *os.Root, name string) error {
	dir, err := root.Open(name)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func publishCleanup(root *os.Root, digest string) (resultErr error) {
	name := ".meca-cleanup-" + rand.Text()
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Remove(name)) }()
	_, writeErr := file.WriteString(digest + "\n")
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	if err := root.Link(name, cleanupName); err != nil {
		return err
	}
	return syncDirectory(root, ".")
}

func finishCleanup(ctx context.Context, root *os.Root, digest string, clearCredentials func(context.Context) error) error {
	if clearCredentials == nil {
		return faults.Errorf(diagcatalog.InstallOSCredentialCleanupIsRequiredRerunMeca)
	}
	if err := checkInstalledBinary(root, digest); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// The marker is durable before removal. A crash or partial OS deletion can
	// be retried without decrypting state or deleting keys for a retained DB.
	if err := root.RemoveAll(statefs.DirectoryName); err != nil {
		return err
	}
	if err := syncDirectory(root, "."); err != nil {
		return err
	}
	if err := clearCredentials(ctx); err != nil {
		return faults.Errorf(diagcatalog.InstallPrivateStateRemovedRerunMecaUninstallTo, err)
	}
	// Keys are gone. Complete removal even if cancellation arrives now.
	if err := checkInstalledBinary(root, digest); err != nil {
		return err
	}
	if err := root.Remove(binaryName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := syncDirectory(root, ".local/bin"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := root.Remove(cleanupName); err != nil {
		return err
	}
	return syncDirectory(root, ".")
}

func checkNoPending(root *os.Root) error {
	_, exists, err := pendingCleanup(root)
	if err != nil {
		return err
	}
	if exists {
		return faults.Errorf(diagcatalog.InstallCredentialCleanupIsUnfinishedRunMecaUninstall)
	}
	return nil
}

// CheckReady prevents normal startup from recreating state during key cleanup.
func CheckReady(home string) (resultErr error) {
	root, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	return checkNoPending(root)
}
