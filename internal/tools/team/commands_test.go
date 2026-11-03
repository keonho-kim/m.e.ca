// Purpose: Verify settled commands release only their owned workspace lease without runtime polling.
// Pattern: Deterministic resource-lifecycle tests.
// Usage: go test -race ./internal/tools/team -run TestCommandLease.
// Related: internal/tools/team/commands.go, internal/tools/team/workspace/workspace.go.
package team

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

type commandLifecycle struct {
	Local
	done map[string]chan struct{}
}

func (l commandLifecycle) CommandDone(id string) (<-chan struct{}, error) { return l.done[id], nil }
func (l commandLifecycle) StopCommands()                                  {}

func TestCommandLeaseSettlesWithoutCollection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	w := workspace.New()
	held, err := w.ReserveShell(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	e := &Executor{Workspace: w, Local: commandLifecycle{done: map[string]chan struct{}{"one": done}}}
	if err := e.holdShell("one", held); err != nil {
		t.Fatal(err)
	}
	// Completion can happen after the runtime starts waiting for a file lease.
	file, err := w.ReserveFiles(ctx, []string{"/project/file"}, func() error { close(done); return nil })
	if err != nil {
		t.Fatal(err)
	}
	file.Release()
	e.StopCommands()
}

func TestCommandLeaseOldCompletionCannotReleaseNewCommand(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		w := workspace.New()
		a, b := make(chan struct{}), make(chan struct{})
		e := &Executor{Workspace: w, Local: commandLifecycle{done: map[string]chan struct{}{"a": a, "b": b}}}
		first, _ := w.ReserveShell(ctx, nil)
		if err := e.holdShell("a", first); err != nil {
			t.Fatal(err)
		}
		e.releaseShell()
		second, err := w.ReserveShell(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := e.holdShell("b", second); err != nil {
			t.Fatal(err)
		}
		close(a)
		synctest.Wait()
		waiting, stop := context.WithCancel(ctx)
		_, err = w.ReserveFiles(waiting, []string{"/project/file"}, func() error { stop(); return nil })
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("second command lost ownership: %v", err)
		}
		close(b)
		e.StopCommands()
		file, err := w.ReserveFiles(ctx, []string{"/project/file"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		file.Release()
	})
}
