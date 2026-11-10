// Purpose: Verify workspace lending, bounded retirement and joined concurrent shutdown.
// Pattern: Deterministic resource-owner tests with controlled close boundaries.
// Usage: go test -race ./internal/commands/server -run TestWorkspaces.
// Related: internal/commands/server/workspaces.go, internal/commands/server/workspaces_lifecycle.go.
package server

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWorkspacesConcurrentCloseJoinsEveryRuntime(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	expected := errors.New("fixture close result")
	w, err := NewWorkspaces(t.Context(), WorkspaceOptions{Default: t.TempDir(), Check: func(p string) (string, error) { return p, nil }, Save: func(context.Context, string) error { return nil }, Open: func(context.Context, Workspace) (WorkspaceRuntime, error) {
		return WorkspaceRuntime{Service: &Service{}, Close: func() error { calls.Add(1); close(entered); <-release; return expected }}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, done, err := w.Acquire(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	done()
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { first <- w.Close() }()
	<-entered
	go func() { second <- w.Close() }()
	close(release)
	if err := <-first; !errors.Is(err, expected) {
		t.Fatal(err)
	}
	if err := <-second; !errors.Is(err, expected) {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("runtime closed more than once")
	}
	if _, _, err := w.Acquire(t.Context(), ""); err == nil {
		t.Fatal("closed registry lent a runtime")
	}
}

func TestWorkspacesShareOpeningAndPinUncertainState(t *testing.T) {
	base := t.TempDir()
	var opened, closed atomic.Int32
	w, err := NewWorkspaces(t.Context(), WorkspaceOptions{Default: base, Check: func(p string) (string, error) { return p, nil }, Save: func(context.Context, string) error { return nil }, Open: func(context.Context, Workspace) (WorkspaceRuntime, error) {
		opened.Add(1)
		return WorkspaceRuntime{Service: &Service{}, Close: func() error { closed.Add(1); return nil }}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			_, done, err := w.Acquire(t.Context(), "")
			if err != nil {
				t.Error(err)
				return
			}
			done()
		})
	}
	workers.Wait()
	if opened.Load() != 1 {
		t.Fatal("duplicate workspace opens", opened.Load())
	}
	// Registered paths need not be opened merely to populate navigation metadata.
	for i := range 7 {
		path := filepath.Join(base, string(rune('a'+i)))
		id := "workspace_" + ProjectScope(path)
		w.mu.Lock()
		w.entries[id] = &workspaceEntry{value: Workspace{ID: id, Path: path}}
		w.order = append(w.order, id)
		w.mu.Unlock()
		service, done, err := w.Acquire(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			service.executionErr = errors.New("uncertain effects")
		}
		done()
	}
	w.retireIdle()
	if closed.Load() != 2 {
		t.Fatal("idle working set not bounded", closed.Load())
	}
	id := "workspace_" + ProjectScope(filepath.Join(base, "a"))
	w.mu.Lock()
	pinned := w.entries[id].runtime.Service != nil
	w.mu.Unlock()
	if !pinned {
		t.Fatal("uncertain execution was evicted and could lose its safety gate")
	}
}

func TestSharedAttachmentAllowanceAndCapacity(t *testing.T) {
	images := NewAttachmentPool()
	release, ok := images.reserve(attachmentBytes - 1)
	if !ok {
		t.Fatal("first reservation failed")
	}
	if _, ok := images.reserve(2); ok {
		t.Fatal("process image byte budget exceeded")
	}
	release()
	release()
	var releases []func()
	for range 16 {
		release, ok := images.reserve(1)
		if !ok {
			t.Fatal("early image limit")
		}
		releases = append(releases, release)
	}
	if _, ok := images.reserve(1); ok {
		t.Fatal("image count budget exceeded")
	}
	for _, release := range releases {
		release()
	}
	if images.count != 0 || images.bytes != 0 {
		t.Fatal("image reservation retained")
	}
	capacity := NewCapacity(2)
	first, ok := capacity.reserve()
	if !ok {
		t.Fatal("first slot")
	}
	second, ok := capacity.reserve()
	if !ok {
		t.Fatal("second slot")
	}
	if _, ok := capacity.reserve(); ok {
		t.Fatal("shared concurrency exceeded")
	}
	first()
	first()
	second()
	if capacity.Active() != 0 {
		t.Fatal("slot retained")
	}
}
