// Purpose: Verify cancellation and actual-path conflict exclusion in a shared workspace.
// Pattern: Deterministic resource admission tests.
// Usage: go test -race ./internal/tools/team.
// Related: internal/tools/team/workspace/workspace.go, internal/tools/team/local.go.
package workspace

import (
	"context"
	"errors"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceDetectsHardlinkAndParentConflicts(t *testing.T) {
	directory := t.TempDir()
	a, b := filepath.Join(directory, "one"), filepath.Join(directory, "alias")
	if err := os.WriteFile(a, []byte("value"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(a, b); err != nil {
		t.Fatal(err)
	}
	for _, second := range []string{b, directory} {
		w := New()
		first, err := w.reserve(context.Background(), []string{a}, false, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		waiting := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			held, err := w.reserve(ctx, []string{second}, false, nil, func() error { close(waiting); return nil })
			if held != nil {
				held.Release()
			}
			done <- err
		}()
		<-waiting
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("overlap did not wait: %v", err)
		}
		first.Release()
	}
}
func TestWorkspaceShellExcludesWritesUntilReleased(t *testing.T) {
	w := New()
	shell, err := w.reserve(context.Background(), nil, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	waiting := make(chan struct{})
	done := make(chan *Lease, 1)
	go func() {
		held, err := w.reserve(context.Background(), []string{filepath.Join(t.TempDir(), "file")}, false, nil, func() error { close(waiting); return nil })
		if err != nil {
			t.Error(err)
		}
		done <- held
	}()
	<-waiting
	shell.Release()
	shell.Release()
	held := <-done
	held.Release()
}

func TestWorkspaceRejectsChangedPreparedAlias(t *testing.T) {
	directory := t.TempDir()
	first, second := filepath.Join(directory, "first"), filepath.Join(directory, "second")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	w := New()
	held, err := w.reserve(context.Background(), []string{first}, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	proposal := toolapi.Approval{Candidate: &toolapi.Candidate{Files: []toolapi.FileDelta{{Path: second}}}}
	var failure *toolapi.Failure
	if err := held.Check(proposal); !errors.As(err, &failure) || failure.Code != toolapi.StaleTarget {
		t.Fatalf("changed alias accepted: %v", err)
	}
}
