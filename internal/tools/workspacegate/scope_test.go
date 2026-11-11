// Purpose: Verify disjoint writes progress while ancestors and unrestricted commands remain exclusive.
// Pattern: Deterministic lease concurrency regression tests.
// Usage: go test -race ./internal/tools/workspacegate.
// Related: internal/tools/workspacegate/gate.go, internal/commands/server/file_effect.go.
package workspacegate

import (
	"context"
	"testing"
	"testing/synctest"
)

func TestDisjointScopesDoNotBlockAndBroadWaitersKeepPriority(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var gate Gate
		first, endFirst := BindPaths(t.Context(), &gate, []string{"/project/a"})
		defer endFirst()
		if err := hold(t, first); err != nil {
			t.Fatal(err)
		}
		second, endSecond := BindPaths(t.Context(), &gate, []string{"/project/b"})
		defer endSecond()
		secondHold, err := Acquire(second)
		if err != nil {
			t.Fatal(err)
		}
		secondHold()
		broad, endBroad := Bind(t.Context(), &gate)
		defer endBroad()
		admitted := make(chan error, 1)
		go func() { admitted <- hold(t, broad) }()
		synctest.Wait()
		ctx, cancel := context.WithCancel(second)
		defer cancel()
		later := make(chan error, 1)
		go func() { later <- hold(t, ctx) }()
		synctest.Wait()
		select {
		case <-later:
			t.Fatal("new scope bypassed an earlier unrestricted command")
		default:
		}
		cancel()
		if err := <-later; err != context.Canceled {
			t.Fatal(err)
		}
		endFirst()
		if err := <-admitted; err != nil {
			t.Fatal(err)
		}
	})
}

func TestAncestorAndChildScopesConflict(t *testing.T) {
	for _, paths := range [][2]string{{"/project/a", "/project/a/child"}, {"/project/a/child", "/project/a"}} {
		synctest.Test(t, func(t *testing.T) {
			var gate Gate
			owner, end := BindPaths(t.Context(), &gate, []string{paths[0]})
			defer end()
			if err := hold(t, owner); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			other, stop := BindPaths(ctx, &gate, []string{paths[1]})
			defer stop()
			done := make(chan error, 1)
			go func() { done <- hold(t, other) }()
			synctest.Wait()
			cancel()
			if err := <-done; err != context.Canceled {
				t.Fatalf("overlapping scope entered: %v", err)
			}
		})
	}
}
