//go:build linux

// Purpose: Project runtime trees read-only while omitting inherited nested mounts.
// Pattern: Bounded filesystem projection; never recursively imports host mounts.
// Usage: enterRoot uses bindRuntime for explicit system runtime roots.
// Related: internal/tools/local/process/sandbox/linux/mountinfo_linux.go, internal/tools/local/process/sandbox/linux/mounts_linux.go.
package linux

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func bindRuntime(root, path string, points []string, budget *int) error {
	source, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	var nested []string
	for _, point := range points {
		if point != source && within(source, point) {
			nested = append(nested, point)
		}
	}
	return projectRuntime(root, source, path, nested, budget, 0)
}

func projectRuntime(root, source, path string, nested []string, budget *int, depth int) error {
	*budget = *budget - 1
	if *budget < 0 || depth > 128 {
		return fmt.Errorf("runtime projection exceeds its entry/depth limit")
	}
	if len(nested) == 0 {
		return bind(root, binding{path: path, source: source, readOnly: true})
	}
	for _, point := range nested {
		if point == source {
			// Keep an empty placeholder, never the content above or below a locked mount.
			info, err := os.Lstat(source)
			if err != nil {
				return err
			}
			target := filepath.Join(root, path)
			if info.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0400)
			if err != nil {
				return err
			}
			return file.Close()
		}
	}
	if err := os.MkdirAll(filepath.Join(root, path), 0755); err != nil {
		return err
	}
	directory, err := os.Open(source)
	if err != nil {
		return err
	}
	entries, readErr := directory.ReadDir(*budget + 1)
	closeErr := directory.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return errors.Join(readErr, closeErr)
	}
	if closeErr != nil {
		return closeErr
	}
	// ReadDir(n) reports EOF only when no further entries remain.
	if len(entries) > *budget {
		return fmt.Errorf("runtime projection exceeds its entry limit")
	}
	for _, entry := range entries {
		childSource, childPath := filepath.Join(source, entry.Name()), filepath.Join(path, entry.Name())
		var below []string
		for _, point := range nested {
			if within(childSource, point) {
				below = append(below, point)
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if len(below) > 0 {
				return fmt.Errorf("nested mount path changed to a symlink")
			}
			link, err := os.Readlink(childSource)
			if err != nil {
				return err
			}
			if err := os.Symlink(link, filepath.Join(root, childPath)); err != nil {
				return err
			}
			*budget = *budget - 1
			continue
		}
		if err := projectRuntime(root, childSource, childPath, below, budget, depth+1); err != nil {
			return err
		}
	}
	return nil
}
