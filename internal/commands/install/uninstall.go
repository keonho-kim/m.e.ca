// Purpose: Remove MECA-owned installation artifacts, credentials and private state.
// Pattern: Uninstallation use case.
// Usage: cmd/meca invokes Uninstall before bootstrap, SQLite or terminal setup.
// Related: internal/commands/install/state.go, internal/commands/install/shell.go, internal/config/credentials/keyring.go.
package install

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"os"
	"path/filepath"
)

func Uninstall(ctx context.Context, home string, clearCredentials func() error) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	info, err := root.Lstat(statefs.DirectoryName)
	if errors.Is(err, os.ErrNotExist) {
		if _, binaryErr := root.Lstat(binaryName); binaryErr == nil {
			return fmt.Errorf("installation state is missing; run meca install before meca uninstall")
		} else if !errors.Is(binaryErr, os.ErrNotExist) {
			return binaryErr
		}
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("application state must be a real directory")
	}
	state, err := statefs.Open(filepath.Join(home, statefs.DirectoryName))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, state.Close()) }()
	lease, err := statefs.Lease(state, true)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	r, err := loadReceipt(root)
	if err != nil {
		return err
	}
	if err := checkInstalledBinary(root, r.Digest); err != nil {
		return err
	}
	var names []string
	for _, registration := range r.Profiles {
		names = append(names, registration.Name)
	}
	profiles, err := planProfiles(root, names, r.Profiles, true)
	if err != nil {
		return err
	}
	if _, err := state.Lstat("credentials.used"); err == nil {
		if clearCredentials == nil {
			return fmt.Errorf("credential cleanup is required")
		}
		if err := clearCredentials(); err != nil {
			return fmt.Errorf("remove MECA credentials before deleting state: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, profile := range profiles {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writeProfile(root, profile, true); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := root.Remove(binaryName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return root.RemoveAll(statefs.DirectoryName)
}
