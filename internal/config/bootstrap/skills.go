// Purpose: Publish self-contained default skill bundles while preserving existing user files.
// Pattern: Embedded asset publication at the startup filesystem boundary.
// Usage: Initialize publishes missing manifests, references and notices through the same private boundary.
// Related: internal/config/bootstrap/bootstrap.go, internal/config/bootstrap/defaults, internal/config/statefs/files.go.
package bootstrap

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path"

	"github.com/keonho-kim/meca/internal/config/statefs"
)

//go:embed defaults
var defaultSkillFiles embed.FS

func bootstrapSkills(root *os.Root) (err error) {
	directory, err := openPrivateDirectory(root, "skills")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, directory.Close()) }()
	return publishSkillDirectory(directory, "defaults")
}

func publishSkillDirectory(target *os.Root, source string) error {
	entries, err := fs.ReadDir(defaultSkillFiles, source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			child, err := openPrivateDirectory(target, name)
			if err != nil {
				return err
			}
			writeErr := publishSkillDirectory(child, path.Join(source, name))
			if err := errors.Join(writeErr, child.Close()); err != nil {
				return err
			}
			continue
		}
		// Existing files are user-owned. Check before copying embedded bytes;
		// PublishMissing still resolves races at the atomic publication boundary.
		if _, err := target.Lstat(name); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		body, err := defaultSkillFiles.ReadFile(path.Join(source, name))
		if err != nil {
			return err
		}
		if err := statefs.PublishMissing(target, name, body); err != nil {
			return err
		}
	}
	return nil
}
