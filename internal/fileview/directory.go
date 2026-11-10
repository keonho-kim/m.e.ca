// Purpose: List bounded, sorted directory pages with Scope-filtered entries.
// Pattern: Filesystem read Adapter.
// Usage: File-browser workers call Reader.Directory through Source.
// Related: internal/fileview/reader.go, internal/fileview/references.go.
package fileview

import (
	"context"
	"errors"

	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

type Entry struct {
	Name      string `json:"name"`
	Directory bool   `json:"directory"`
}

func (executor *Reader) Directory(ctx context.Context, settings policy.Settings, path string, offset, limit int, revision string) (DirectoryPage, error) {
	if offset < 0 || offset > 100000 || limit < 1 || limit > 1000 || len(revision) > 64 {
		return DirectoryPage{}, faults.Errorf(diagcatalog.FileviewInvalidDirectoryPage)
	}
	resolved, err := executor.resolve(settings.Scope, path)
	if err != nil {
		return DirectoryPage{}, err
	}
	if err := settings.Check(); err != nil {
		return DirectoryPage{}, err
	}
	all, version, err := executor.directories.load(ctx, resolved, revision)
	if err != nil {
		return DirectoryPage{}, err
	}
	if offset >= len(all) {
		return DirectoryPage{Revision: version}, nil
	}
	entries := all[offset:min(len(all), offset+limit)]
	visible := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		target, resolveErr := executor.resolve(settings.Scope, filepath.Join(resolved.Absolute, entry.Name))
		if resolveErr == nil {
			info, statErr := target.Root.Stat(target.Relative)
			if statErr == nil {
				entry.Directory = info.IsDir()
				visible = append(visible, entry)
			}
		}
	}
	return DirectoryPage{Entries: visible, More: len(all) > offset+limit, Revision: version}, nil
}

func readDirectoryEntries(ctx context.Context, path Target) (entries []Entry, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := path.Root.OpenFile(path.Relative, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch, readErr := file.ReadDir(1000)
		for _, entry := range batch {
			entries = append(entries, Entry{Name: entry.Name(), Directory: entry.IsDir()})
		}
		if len(entries) > 25000 {
			return nil, faults.Errorf(diagcatalog.FileviewDirectoryExceedsEntries)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Directory != entries[j].Directory {
			return entries[i].Directory
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}
