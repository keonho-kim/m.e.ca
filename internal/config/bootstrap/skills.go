// Purpose: Publish the bundled skill tree with startup or explicit refresh semantics.
// Pattern: Embedded asset publication at the startup filesystem boundary.
// Usage: Initialize preserves existing files; RefreshSkills replaces owned release files under the same lock.
// Related: internal/config/bootstrap/bootstrap.go, internal/config/bootstrap/refresh_skills.go, internal/config/statefs/files.go.
package bootstrap

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"os"
	"path"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

//go:embed defaults
var defaultSkillFiles embed.FS

type skillDirectory struct {
	name, source string
	files        []string
	children     []skillDirectory
}

func bootstrapSkills(ctx context.Context, root *os.Root, refresh bool) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	directory, created, err := openPrivateDirectory(root, "skills")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, directory.Close()) }()
	lease, err := statefs.Lock(ctx, directory)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	// The name may have changed while this process waited for the directory lease.
	current, err := root.Lstat("skills")
	if err != nil {
		return err
	}
	locked, err := lease.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(current, locked) {
		return faults.Errorf(diagcatalog.BootstrapChangedDuringInitialization, "skills")
	}
	bundle, err := readSkillDirectory("", "defaults")
	if err != nil {
		return err
	}
	if err := bundle.checkExisting(ctx, directory, "", refresh); err != nil {
		return err
	}
	if err := bundle.publish(ctx, directory, refresh); err != nil {
		return err
	}
	if created {
		return errors.Join(ctx.Err(), statefs.SyncDirectory(root))
	}
	return ctx.Err()
}

func readSkillDirectory(name, source string) (skillDirectory, error) {
	bundle := skillDirectory{name: name, source: source}
	entries, err := fs.ReadDir(defaultSkillFiles, source)
	if err != nil {
		return bundle, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			child, err := readSkillDirectory(name, path.Join(source, name))
			if err != nil {
				return bundle, err
			}
			bundle.children = append(bundle.children, child)
			continue
		}
		bundle.files = append(bundle.files, name)
	}
	return bundle, nil
}

// checkExisting detects a pre-existing conflict before publishing sibling assets.
// Descendants of missing directories cannot be present.
func (d skillDirectory) checkExisting(ctx context.Context, target *os.Root, relative string, refresh bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, child := range d.children {
		name := path.Join(relative, child.name)
		info, err := target.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return faults.Errorf(diagcatalog.BootstrapMustBeARealDirectory, path.Join("skills", name))
		}
		if err := child.checkExisting(ctx, target, name, refresh); err != nil {
			return err
		}
	}
	if refresh {
		for _, file := range d.files {
			name := path.Join(relative, file)
			info, err := target.Lstat(name)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return faults.New(diagcatalog.FailedPrecondition, diagcatalog.BootstrapSkillFileMustBeRegular, name)
			}
		}
	}
	return nil
}

func (d skillDirectory) publish(ctx context.Context, target *os.Root, refresh bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	createdChild := false
	for _, child := range d.children {
		directory, created, err := openPrivateDirectory(target, child.name)
		if err != nil {
			return err
		}
		createdChild = createdChild || created
		writeErr := child.publish(ctx, directory, refresh)
		if err := errors.Join(writeErr, directory.Close()); err != nil {
			return err
		}
	}
	read := func(name string) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return defaultSkillFiles.ReadFile(path.Join(d.source, name))
	}
	if !refresh {
		return statefs.PublishMissingFiles(target, d.files, read, createdChild)
	}
	for _, name := range d.files {
		content, err := read(name)
		if err != nil {
			return err
		}
		if err := refreshSkillFile(target, name, content); err != nil {
			return err
		}
	}
	if createdChild {
		return statefs.SyncDirectory(target)
	}
	return ctx.Err()
}
