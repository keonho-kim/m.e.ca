// Purpose: Reclaim only bounded old diagnostic directories whose process lease has ended.
// Pattern: Process-safe archive retention.
// Usage: OpenFile holds the logs-directory lock while inspecting and removing archives.
// Related: internal/diagnostics/logfile/file.go, internal/config/statefs/lease_unix.go.
package logfile

import (
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"

	"github.com/keonho-kim/meca/internal/config/statefs"
)

func retainLogs(parent *os.Root) error {
	file, err := parent.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := file.ReadDir(4097)
	if errors.Is(readErr, io.EOF) {
		readErr = nil
	}
	err = errors.Join(readErr, file.Close())
	if err != nil {
		return err
	}
	// Preserve unknown contents and decline an unexpectedly large log directory.
	if len(entries) > 4096 {
		return fault.Errorf(catalog.DiagnosticDirectoryLimit)
	}
	type archive struct {
		name string
		at   time.Time
	}
	var old []archive
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "run-") || len(entry.Name()) != 30 {
			continue
		}
		root, err := parent.OpenRoot(entry.Name())
		if err != nil {
			continue
		}
		lease, err := statefs.Lease(root, true)
		if err != nil {
			_ = root.Close()
			continue
		}
		at, known := archiveTime(root)
		if known {
			old = append(old, archive{entry.Name(), at})
		}
		if err := errors.Join(lease.Close(), root.Close()); err != nil {
			return err
		}
	}
	sort.Slice(old, func(i, j int) bool { return old[i].at.After(old[j].at) })
	for _, entry := range old[min(len(old), retainedRuns):] {
		root, err := parent.OpenRoot(entry.name)
		if err != nil {
			continue
		}
		lease, err := statefs.Lease(root, true)
		if err != nil {
			_ = root.Close()
			continue
		}
		err = errors.Join(parent.RemoveAll(entry.name), lease.Close(), root.Close())
		if err != nil {
			return err
		}
	}
	return nil
}

// Rotation can stop between renaming current and creating its replacement.
// Only recognize our own files; retain directories with unknown contents.
func archiveTime(root *os.Root) (time.Time, bool) {
	file, err := root.Open(".")
	if err != nil {
		return time.Time{}, false
	}
	entries, err := file.ReadDir(4)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	err = errors.Join(err, file.Close())
	if err != nil || len(entries) > 3 {
		return time.Time{}, false
	}
	var latest time.Time
	for _, entry := range entries {
		switch entry.Name() {
		case "current.jsonl", "previous-1.jsonl", "previous-2.jsonl":
		default:
			return time.Time{}, false
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return time.Time{}, false
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	if latest.IsZero() {
		info, err := root.Stat(".")
		if err != nil {
			return time.Time{}, false
		}
		latest = info.ModTime()
	}
	return latest, true
}
