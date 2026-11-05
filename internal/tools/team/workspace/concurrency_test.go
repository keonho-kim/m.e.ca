// Purpose: Verify independent command targets coexist with bounded file work and conflicting aliases wait.
// Pattern: Deterministic admission tests with cancellable conflict probes.
// Usage: go test -race ./internal/tools/team/workspace -run 'TestIndependent|TestCommand'.
// Related: internal/tools/team/workspace/workspace.go, internal/tools/team/workspace/reservation.go.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestIndependentCommandTargetsDoNotConsumeFileSlots(t *testing.T) {
	w := New()
	root := t.TempDir()
	var leases []*Lease
	defer func() {
		for _, held := range leases {
			held.Release()
		}
		if len(w.writes) != 0 || w.files != 0 {
			t.Fatal("released operations retained target reservations")
		}
	}()
	for i := range 128 {
		held, err := w.ReserveCommand(context.Background(), []string{filepath.Join(root, fmt.Sprintf("command-%d", i))}, nil, func() error {
			return fmt.Errorf("independent command was serialized")
		})
		if err != nil {
			t.Fatal(err)
		}
		leases = append(leases, held)
	}
	for i := range 16 {
		held, err := w.ReserveFiles(context.Background(), []string{filepath.Join(root, fmt.Sprintf("file-%d", i))}, nil, func() error {
			return fmt.Errorf("command targets consumed file capacity")
		})
		if err != nil {
			t.Fatal(err)
		}
		leases = append(leases, held)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	held, err := w.ReserveFiles(ctx, []string{filepath.Join(root, "extra-file")}, nil, func() error { cancel(); return nil })
	if held != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("file reservation capacity was lost", err)
	}
	before := len(w.writes)
	if held, err := w.ReserveCommand(context.Background(), nil, nil, nil); held != nil || err != nil || len(w.writes) != before {
		t.Fatal("a command without known mutation targets retained a lease", err)
	}
	if allocs := testing.AllocsPerRun(100, func() {
		_, _ = w.ReserveCommand(context.Background(), nil, nil, nil)
	}); allocs != 0 {
		t.Fatalf("a command without known mutation targets allocated: %v", allocs)
	}
}

func TestCommandTargetsSerializeHardlinksAndParents(t *testing.T) {
	root := t.TempDir()
	path, alias := filepath.Join(root, "file"), filepath.Join(root, "alias")
	if err := os.WriteFile(path, []byte("value"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(path, alias); err != nil {
		t.Fatal(err)
	}
	w := New()
	first, err := w.ReserveCommand(context.Background(), []string{path}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()
	for _, target := range []string{alias, root} {
		ctx, cancel := context.WithCancel(context.Background())
		waited := false
		held, err := w.ReserveCommand(ctx, []string{target}, nil, func() error { waited = true; cancel(); return nil })
		cancel()
		if held != nil || !waited || !errors.Is(err, context.Canceled) {
			t.Fatal("conflicting command did not wait on the actual target", err)
		}
	}
}
