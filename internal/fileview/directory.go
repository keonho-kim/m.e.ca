// Purpose: List bounded, sorted directory pages with Scope-filtered entries.
// Pattern: Filesystem read Adapter.
// Usage: File-browser workers call Reader.Directory through Source.
// Related: internal/fileview/reader.go, internal/fileview/references.go.
package fileview

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/policy"
	"io"
	"path/filepath"
	"sort"
)

type Entry struct {
	Name      string `json:"name"`
	Directory bool   `json:"directory"`
}

func (executor *Reader) Directory(ctx context.Context, settings policy.Settings, path string, offset, limit int) ([]Entry, bool, error) {
	resolved, err := executor.resolve(settings.Scope, path)
	if err != nil {
		return nil, false, err
	}
	if err := settings.Check(); err != nil {
		return nil, false, err
	}
	entries, more, err := readDirectory(ctx, resolved, offset, limit)
	if err != nil {
		return nil, false, err
	}
	visible := entries[:0]
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
	return visible, more, nil
}

func readDirectory(ctx context.Context, path Target, offset, limit int) (entries []Entry, more bool, err error) {
	if offset < 0 || offset > 100000 || limit < 1 || limit > 1000 {
		return nil, false, fmt.Errorf("invalid directory page")
	}
	file, err := path.Root.Open(path.Relative)
	if err != nil {
		return nil, false, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()

	for {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		batch, readErr := file.ReadDir(1000)
		for _, entry := range batch {
			entries = append(entries, Entry{Name: entry.Name(), Directory: entry.IsDir()})
		}
		if len(entries) > 25000 {
			return nil, false, fmt.Errorf("directory exceeds 25000 entries")
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, false, readErr
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Directory != entries[j].Directory {
			return entries[i].Directory
		}
		return entries[i].Name < entries[j].Name
	})
	if offset >= len(entries) {
		return nil, false, nil
	}
	more = len(entries) > offset+limit
	entries = entries[offset:min(len(entries), offset+limit)]
	return entries, more, nil
}
