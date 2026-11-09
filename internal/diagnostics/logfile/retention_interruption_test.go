// Purpose: Prevent orphaned diagnostic archives after rotation.
// Pattern: Resource ownership regression test.
// Usage: go test ./internal/diagnostics.
// Related: internal/diagnostics/logfile/retention.go.
package logfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInterruptedRotationRetention(t *testing.T) {
	directory := t.TempDir()
	for i := 0; i < retainedRuns+3; i++ {
		file, err := OpenFile(t.Context(), directory)
		if err != nil {
			t.Fatal(err)
		}
		name := file.name
		if _, err := file.Write([]byte("old record\n")); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(directory, name, "current.jsonl"), filepath.Join(directory, name, "previous-1.jsonl")); err != nil {
			t.Fatal(err)
		}
	}
	current, err := OpenFile(t.Context(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer current.Close()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "run-") {
			count++
		}
	}
	if count > retainedRuns+1 {
		t.Fatalf("interrupted rotation directories evade retention: %d directories, limit %d", count, retainedRuns+1)
	}
}
