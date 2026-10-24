// Purpose: Resolve project-owned session directories and enumerate database filenames without a registry.
// Pattern: Filesystem discovery boundary.
// Usage: Create and Open address one project; global resume visits each project directory.
// Related: internal/storage/session/project.go, internal/storage/sqlite/catalog.go, internal/storage/sqlite/conversations.go.
package sqlite

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/keonho-kim/meca/internal/config/statefs"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (s *Sessions) projectDirectory(project string, create bool) (directory, canonical string, err error) {
	if err := sessionstate.CheckProject(project, project); err != nil {
		return "", "", err
	}
	canonical, err = filepath.EvalSymlinks(project)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("session project must be a directory")
	}
	key, err := sessionstate.ProjectKey(canonical)
	if err != nil {
		return "", "", err
	}
	directory = filepath.Join(s.directory, key)
	if !create {
		if _, err := os.Lstat(directory); err != nil {
			return "", "", err
		}
	}
	root, err := statefs.Open(directory)
	if err != nil {
		return "", "", err
	}
	return directory, canonical, root.Close()
}

func (s *Sessions) walkDatabases(ctx context.Context, project string, visit func(string, string, string) error) error {
	if project != "" {
		directory, _, err := s.projectDirectory(project, false)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		return walkDatabaseDirectory(ctx, directory, visit)
	}
	return walkDirectory(ctx, s.directory, func(entry os.DirEntry) error {
		name := entry.Name()
		index := strings.LastIndexByte(name, '-')
		if index < 1 || len(name)-index-1 != 32 {
			return nil
		}
		if _, err := hex.DecodeString(name[index+1:]); err != nil {
			return nil
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("project session group must be a real directory: %s", name)
		}
		directory := filepath.Join(s.directory, name)
		root, err := statefs.Open(directory)
		if err != nil {
			return err
		}
		if err := root.Close(); err != nil {
			return err
		}
		return walkDatabaseDirectory(ctx, directory, visit)
	})
}

func walkDatabaseDirectory(ctx context.Context, directory string, visit func(string, string, string) error) error {
	return walkDirectory(ctx, directory, func(entry os.DirEntry) error {
		name := entry.Name()
		if !strings.HasPrefix(name, "session_") || !strings.HasSuffix(name, ".db") {
			return nil
		}
		id := strings.TrimSuffix(name, ".db")
		if !sessionstate.ValidID(id) {
			return fmt.Errorf("invalid session database filename: %s", name)
		}
		return visit(filepath.Join(directory, name), id, filepath.Base(directory))
	})
}

func walkDirectory(ctx context.Context, path string, visit func(os.DirEntry) error) (err error) {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, directory.Close()) }()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, readErr := directory.ReadDir(128)
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := visit(entry); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}
