// Purpose: Verify native selection bounds and cancellation without opening an unattended desktop dialog.
// Pattern: Deterministic adapter-boundary regression tests.
// Usage: go test ./internal/desktop/folderpicker.
// Related: internal/desktop/folderpicker/picker.go, internal/serverapi/ui_folder_test.go.
package folderpicker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectionRequiresExistingAbsoluteDirectory(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "document")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"relative", file, filepath.Join(root, "missing"), strings.Repeat("a", 4097), "/invalid\x00path", "/\xff"} {
		if _, err := selectedFolder(path); err == nil {
			t.Fatal("invalid directory selection accepted")
		}
	}
	if got, err := selectedFolder(root + string(filepath.Separator)); err != nil || got != root {
		t.Fatal(got, err)
	}
	if got, err := selectedFolder(""); err != nil || got != "" {
		t.Fatal("cancel is not an empty selection", got, err)
	}
}
func TestCanceledAndSSHRequestsDoNotOpenDesktop(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Pick(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	t.Setenv("SSH_CONNECTION", "remote fixture")
	if _, err := Pick(t.Context()); err == nil {
		t.Fatal("SSH forwarded request opened a desktop")
	}
}
func TestNativeOutputCannotGrowPastItsBound(t *testing.T) {
	out := &pathOutput{}
	if _, err := out.Write(make([]byte, 16<<10)); err != nil {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte("overflow")); err == nil || len(out.body) != 16<<10 {
		t.Fatal(len(out.body), err)
	}
}
