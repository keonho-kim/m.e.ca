// Purpose: Install MECA's binary, refresh release prompts and register the user's shell PATH.
// Pattern: Installation use case.
// Usage: cmd/meca invokes Install before terminal startup.
// Related: internal/commands/install/binary.go, internal/commands/install/shell.go, internal/config/prompts/refresh.go, internal/config/bootstrap/bootstrap.go.
package install

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"os"
	"path/filepath"
	"slices"
)

type Options struct{ Home, Executable, Shell, Path string }
type Installed struct {
	Binary       string
	RestartShell bool
}

func Install(ctx context.Context, options Options) (result Installed, err error) {
	if err := ctx.Err(); err != nil {
		return result, err
	}
	root, err := os.OpenRoot(options.Home)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	state, err := statefs.Open(filepath.Join(options.Home, statefs.DirectoryName))
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, state.Close()) }()
	lease, err := statefs.Lease(state, true)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	previous, err := loadReceipt(root)
	if err != nil {
		return result, err
	}
	if err := checkInstalledBinary(root, previous.Digest); err != nil {
		return result, err
	}
	digest, err := binaryDigest(options.Executable)
	if err != nil {
		return result, err
	}
	result.Binary = BinaryPath(options.Home)
	result.RestartShell = !slices.Contains(filepath.SplitList(options.Path), filepath.Dir(result.Binary))
	var names []string
	for _, r := range previous.Profiles {
		names = append(names, r.Name)
	}
	if result.RestartShell {
		selected, err := profileNames(root, options.Shell)
		if err != nil {
			return result, err
		}
		for _, name := range selected {
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	profiles, err := planProfiles(root, names, previous.Profiles, false)
	if err != nil {
		return result, err
	}
	if err := prompts.Refresh(state); err != nil {
		return result, err
	}
	if _, err := bootstrap.Initialize(options.Home); err != nil {
		return result, err
	}
	next := receipt{Digest: digest}
	for _, profile := range profiles {
		next.Profiles = append(next.Profiles, profile.registration)
	}
	// Record ownership first so interrupted installation remains removable.
	if err := saveReceipt(options.Home, next); err != nil {
		return result, err
	}
	if err := copyBinary(root, options.Executable, digest); err != nil {
		return result, err
	}
	for _, profile := range profiles {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := writeProfile(root, profile, false); err != nil {
			return result, err
		}
	}
	return result, nil
}
