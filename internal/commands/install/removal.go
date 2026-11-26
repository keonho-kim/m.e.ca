// Purpose: Complete interrupted removal of private state and the installed executable.
// Pattern: Irreversible uninstall state transition with a durable ownership marker.
// Usage: Uninstall resumes pending removal; startup and Install reject an unfinished removal.
// Related: internal/commands/install/uninstall.go, internal/commands/install/install.go, cmd/meca/main.go.
package install

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"os"
	"strings"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const removalName = ".meca-uninstall"

func pendingRemoval(root *os.Root) (string, bool, error) {
	info, err := root.Lstat(removalName)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() != 65 {
		return "", false, faults.Errorf(diagcatalog.InstallInvalidPrivateRemovalMarker)
	}
	body, _, _, err := readFile(root, removalName)
	if err != nil {
		return "", false, err
	}
	if len(body) != 65 {
		return "", false, faults.Errorf(diagcatalog.InstallRemovalMarkerChangedWhileReading)
	}
	digest := string(body[:64])
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 || body[64] != '\n' {
		return "", false, faults.Errorf(diagcatalog.InstallInvalidRemovalIdentity)
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

func publishRemoval(root *os.Root, digest string) (resultErr error) {
	// Direct execution can create state before an installation receipt exists.
	if digest == "" {
		digest = strings.Repeat("0", 64)
	}
	name := ".meca-uninstall-" + rand.Text()
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Remove(name)) }()
	_, writeErr := file.WriteString(digest + "\n")
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	if err := root.Link(name, removalName); err != nil {
		return err
	}
	return syncDirectory(root, ".")
}

func finishRemoval(ctx context.Context, root *os.Root, digest string) error {
	if err := checkInstalledBinary(root, digest); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// The home lease and durable marker prevent startup from recreating state
	// after interrupted removal. All credential material belongs to this state.
	if err := root.RemoveAll(statefs.DirectoryName); err != nil {
		return err
	}
	if err := syncDirectory(root, "."); err != nil {
		return err
	}
	// Private state is gone. Finish removal even if cancellation arrives now.
	if err := checkInstalledBinary(root, digest); err != nil {
		return err
	}
	if err := root.Remove(binaryName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := syncDirectory(root, ".local/bin"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := root.Remove(removalName); err != nil {
		return err
	}
	return syncDirectory(root, ".")
}

func checkNoPending(root *os.Root) error {
	_, exists, err := pendingRemoval(root)
	if err != nil {
		return err
	}
	if exists {
		return faults.Errorf(diagcatalog.InstallRemovalUnfinished)
	}
	return nil
}

// CheckReady prevents normal startup from recreating state during removal.
func CheckReady(home string) (resultErr error) {
	root, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	return checkNoPending(root)
}
