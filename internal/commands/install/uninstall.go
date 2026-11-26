// Purpose: Remove MECA-owned installation artifacts, credentials and private state.
// Pattern: Uninstallation use case.
// Usage: cmd/meca invokes Uninstall before bootstrap, SQLite or terminal setup.
// Related: internal/commands/install/state.go, internal/commands/install/shell.go, internal/config/credentials/persistence.go.
package install

import (
	"context"
	"errors"

	"os"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Uninstall(ctx context.Context, home string) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	homeLease, err := statefs.Lease(root, true)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, homeLease.Close()) }()
	pending, exists, err := pendingRemoval(root)
	if err != nil {
		return err
	}
	info, err := root.Lstat(statefs.DirectoryName)
	if errors.Is(err, os.ErrNotExist) {
		if exists {
			return finishRemoval(ctx, root, pending)
		}
		if _, binaryErr := root.Lstat(binaryName); binaryErr == nil {
			return faults.Errorf(diagcatalog.InstallInstallationStateIsMissingRunMecaInstall)
		} else if !errors.Is(binaryErr, os.ErrNotExist) {
			return binaryErr
		}
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return faults.Errorf(diagcatalog.InstallApplicationStateMustBeARealDirectory)
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
	if exists {
		return finishRemoval(ctx, root, pending)
	}
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
	if err := publishRemoval(root, r.Digest); err != nil {
		return err
	}
	return finishRemoval(ctx, root, r.Digest)
}
