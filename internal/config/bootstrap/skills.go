// Purpose: Publish self-contained default skill bundles while preserving existing user files.
// Pattern: Embedded asset publication at the startup filesystem boundary.
// Usage: Initialize publishes missing manifests, references and notices through the same private boundary.
// Related: internal/config/bootstrap/bootstrap.go, internal/config/bootstrap/defaults, internal/config/statefs/files.go.
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

func bootstrapSkills(ctx context.Context, root *os.Root) (err error) {
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
	if err := bundle.checkExisting(ctx, directory, ""); err != nil {
		return err
	}
	if err := bundle.publish(ctx, directory); err != nil {
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
func (d skillDirectory) checkExisting(ctx context.Context, target *os.Root, relative string) error {
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
		if err := child.checkExisting(ctx, target, name); err != nil {
			return err
		}
	}
	return nil
}

func (d skillDirectory) publish(ctx context.Context, target *os.Root) error {
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
		writeErr := child.publish(ctx, directory)
		if err := errors.Join(writeErr, directory.Close()); err != nil {
			return err
		}
	}
	return statefs.PublishMissingFiles(target, d.files, func(name string) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return defaultSkillFiles.ReadFile(path.Join(d.source, name))
	}, createdChild)
}
