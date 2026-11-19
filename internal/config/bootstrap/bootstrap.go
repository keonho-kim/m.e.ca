// Purpose: Initialize missing private configuration, prompts and skills without replacing user state.
// Pattern: Simple Module (startup I/O boundary).
// Usage: Called before database and terminal startup by cmd/meca.
// Related: internal/config/statefs/files.go, internal/config/bootstrap/skills.go, internal/config/bootstrap/bootstrap_test.go.
package bootstrap

import (
	"context"
	"errors"

	"os"
	"path/filepath"

	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/config/statefs"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/pelletier/go-toml/v2"
)

func Initialize(ctx context.Context, home string) (configuration config.Config, err error) {
	if err := ctx.Err(); err != nil {
		return configuration, err
	}
	directory := filepath.Join(home, statefs.DirectoryName)
	root, err := statefs.Open(directory)
	if err != nil {
		return configuration, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	content, err := toml.Marshal(config.Defaults())
	if err != nil {
		return configuration, err
	}
	if err = statefs.PublishMissing(root, config.Filename, content); err != nil {
		return configuration, err
	}
	info, err := root.Lstat(config.Filename)
	if err != nil {
		return configuration, err
	}
	if !info.Mode().IsRegular() {
		return configuration, faults.Errorf(diagcatalog.BootstrapConfigurationMustBeARegularFile)
	}
	for _, name := range []string{"logs", "sessions"} {
		directory, _, err := openPrivateDirectory(root, name)
		if err != nil {
			return configuration, err
		}
		if err := directory.Close(); err != nil {
			return configuration, err
		}
	}
	if err = prompts.Bootstrap(root); err != nil {
		return configuration, err
	}
	if err = bootstrapSkills(ctx, root); err != nil {
		return configuration, err
	}
	return config.Load(directory)
}

func openPrivateDirectory(parent *os.Root, name string) (*os.Root, bool, error) {
	before, err := parent.Lstat(name)
	created := false
	if errors.Is(err, os.ErrNotExist) {
		mkdirErr := parent.Mkdir(name, 0700)
		if mkdirErr != nil && !errors.Is(mkdirErr, os.ErrExist) {
			return nil, false, mkdirErr
		}
		created = mkdirErr == nil
		before, err = parent.Lstat(name)
	}
	if err != nil {
		return nil, created, err
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, created, faults.Errorf(diagcatalog.BootstrapMustBeARealDirectory, name)
	}
	directory, err := parent.OpenRoot(name)
	if err != nil {
		return nil, created, err
	}
	file, err := directory.Open(".")
	if err != nil {
		return nil, created, errors.Join(err, directory.Close())
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		return nil, created, errors.Join(faults.Errorf(diagcatalog.BootstrapChangedDuringInitialization, name), err, file.Close(), directory.Close())
	}
	const permissionBits = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
	if after.Mode()&permissionBits != 0700 {
		err = file.Chmod(0700)
	}
	if err := errors.Join(err, file.Close()); err != nil {
		return nil, created, errors.Join(err, directory.Close())
	}
	return directory, created, nil
}
