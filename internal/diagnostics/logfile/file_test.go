// Purpose: Verify process-owned private log rotation preserves live writers and bounded archives.
// Pattern: Filesystem lifecycle tests.
// Usage: go test ./internal/diagnostics -run TestLogFile.
// Related: internal/diagnostics/logfile/file.go, internal/diagnostics/logfile/retention.go.
package logfile

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogFileRotationAndLiveOwnerPreservation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	first, err := OpenFile(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	for range 14 {
		if _, err := first.Write(bytes.Repeat([]byte("x"), 1<<20)); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(dir, first.name))
	if err != nil {
		t.Fatal(err)
	}
	bytes := int64(0)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		bytes += info.Size()
		if info.Mode().Perm()&0077 != 0 {
			t.Fatal("non-private log file")
		}
	}
	if len(entries) != 3 || bytes > 3*maxFileBytes {
		t.Fatalf("files=%d bytes=%d", len(entries), bytes)
	}
	for range 12 {
		f, err := OpenFile(t.Context(), dir)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte("record\n")); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := first.Write([]byte("still-live\n")); err != nil {
		t.Fatal("cleanup removed a live process log", err)
	}
	entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "run-") {
			count++
		}
	}
	if count > retainedRuns+2 {
		t.Fatalf("unbounded inactive directories: %d", count)
	}
}
