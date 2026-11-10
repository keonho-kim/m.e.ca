//go:build unix

// Purpose: Reject blocking special files and detach returned pages from directory scan storage.
// Pattern: Filesystem resource regression tests.
// Usage: go test -race ./internal/fileview.
// Related: internal/fileview/preview.go, internal/fileview/directory.go.
package fileview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/policy"
	"golang.org/x/sys/unix"
)

func TestSpecialFileReadDoesNotBlock(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprint(directory), func(t *testing.T) {
			base := t.TempDir()
			path := filepath.Join(base, "pipe")
			if err := unix.Mkfifo(path, 0600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(base)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			reader := New(base, func(policy.Scope, string) (Target, error) {
				return Target{Absolute: path, Relative: "pipe", Root: root}, nil
			}, func(s string) string { return s })
			done := make(chan error, 1)
			go func() {
				if directory {
					_, err := reader.Directory(t.Context(), policy.Settings{Scope: policy.System}, "pipe", 0, 100, "")
					done <- err
				} else {
					_, err := reader.Preview(t.Context(), policy.Settings{Scope: policy.System}, "pipe")
					done <- err
				}
			}()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("special file accepted")
				}
			case <-time.After(200 * time.Millisecond):
				peer, err := os.OpenFile(path, os.O_RDWR|unix.O_NONBLOCK, 0600)
				if err != nil {
					t.Fatal(err)
				}
				<-done
				peer.Close()
				t.Fatal("special-file open ignored cancellation ownership")
			}
		})
	}
}

func TestDirectoryPageOwnsOnlySelectedEntries(t *testing.T) {
	directory := t.TempDir()
	for i := range 300 {
		if err := os.WriteFile(filepath.Join(directory, fmt.Sprintf("entry-%03d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	reader := New(directory, func(policy.Scope, string) (Target, error) {
		return Target{Absolute: directory, Relative: ".", Root: root}, nil
	}, func(s string) string { return s })
	value, err := reader.Directory(context.Background(), policy.Settings{Scope: policy.System}, ".", 20, 10, "")
	page, more := value.Entries, value.More
	if err != nil || !more || len(page) != 10 || page[0].Name != "entry-020" {
		t.Fatal(page, more, err)
	}
	if cap(page) > 2*len(page) {
		t.Fatalf("page retained unrelated entries: len=%d cap=%d", len(page), cap(page))
	}
}
