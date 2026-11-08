// Purpose: Verify cancellation and actual-path conflict exclusion in a shared workspace.
// Pattern: Deterministic resource admission tests.
// Usage: go test -race ./internal/tools/team.
// Related: internal/tools/team/workspace/workspace.go, internal/tools/team/local.go.
package workspace

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
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
func TestWorkspaceCommandExcludesOnlyOverlappingWrites(t *testing.T) {
	w := New()
	path := filepath.Join(t.TempDir(), "file")
	command, err := w.ReserveCommand(context.Background(), []string{path}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	waiting := make(chan struct{})
	done := make(chan *Lease, 1)
	go func() {
		held, err := w.ReserveFiles(context.Background(), []string{path}, nil, func() error { close(waiting); return nil })
		if err != nil {
			t.Error(err)
		}
		done <- held
	}()
	<-waiting
	command.Release()
	command.Release()
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
	var failure *faults.Failure
	if err := held.Check(proposal); !errors.As(err, &failure) || failure.Code != diagcatalog.StaleTarget {
		t.Fatalf("changed alias accepted: %v", err)
	}
}
