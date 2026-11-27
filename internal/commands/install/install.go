// Purpose: Install MECA with private credentials, release prompts and skills, and shell PATH registration.
// Pattern: Installation use case.
// Usage: cmd/meca invokes Install before terminal startup.
// Related: internal/commands/install/binary.go, internal/commands/install/shell.go,
// internal/config/credentials/prepare.go, internal/config/prompts/refresh.go, internal/config/bootstrap/refresh_skills.go.
package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/config/statefs"
)

type Options struct {
	Home, Executable, Shell, Path string
	// PrepareExtension runs after the base installation succeeds while its state is still leased.
	PrepareExtension func(context.Context, string) error
}

type Installed struct {
	// Binary is set after the base installation succeeds, even if an extension fails.
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
	homeLease, err := statefs.Lease(root, true)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, homeLease.Close()) }()
	if err := checkNoPending(root); err != nil {
		return result, err
	}
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
	if err := checkNoPending(root); err != nil {
		return result, err
	}
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
	binary := BinaryPath(options.Home)
	restartShell := !slices.Contains(filepath.SplitList(options.Path), filepath.Dir(binary))
	var names []string
	for _, r := range previous.Profiles {
		names = append(names, r.Name)
	}
	if restartShell {
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
	if err := credentials.Prepare(ctx, state.Name()); err != nil {
		return result, err
	}
	if err := prompts.Refresh(state); err != nil {
		return result, err
	}
	if _, err := bootstrap.Initialize(ctx, options.Home); err != nil {
		return result, err
	}
	if err := bootstrap.RefreshSkills(ctx, state); err != nil {
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
	result = Installed{Binary: binary, RestartShell: restartShell}
	if options.PrepareExtension != nil {
		return result, options.PrepareExtension(ctx, state.Name())
	}
	return result, nil
}
