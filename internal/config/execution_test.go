// Purpose: Verify trusted additional write roots are persisted without introducing model-controlled scope settings.
// Pattern: Configuration persistence test.
// Usage: go test ./internal/config -run TestExecution.
// Related: internal/config/settings.go.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutionWriteRoots(t *testing.T) {
	c := Defaults()
	directory := t.TempDir()
	cache := t.TempDir()
	c.Execution.WritablePaths = []string{cache}
	if err := Save(directory, c); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(directory)
	if err != nil || len(restored.Execution.WritablePaths) != 1 || restored.Execution.WritablePaths[0] != cache {
		t.Fatal(restored, err)
	}
	body, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil || !strings.Contains(string(body), "[execution]") || !strings.Contains(string(body), "writable_paths") {
		t.Fatal(string(body), err)
	}
	c.Execution.WritablePaths = []string{"relative/cache"}
	if err := Save(directory, c); err == nil {
		t.Fatal("relative write root accepted")
	}
}
