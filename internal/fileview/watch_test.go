// Purpose: Verify active-directory observation publishes refreshes and joins owned filesystem resources.
// Pattern: Filesystem observer integration test with explicit readiness and cancellation.
// Usage: go test ./internal/fileview -run TestWatch.
// Related: internal/fileview/watch.go.
package fileview

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestWatchRefreshAndCancellation(t *testing.T) {
	base := t.TempDir()
	root, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	reader := New(base, func(policy.Scope, string) (Target, error) {
		return Target{Absolute: base, Relative: ".", Root: root}, nil
	}, func(v string) string { return v })
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	notifications := make(chan FileChange, 2)
	done := make(chan error, 1)
	go func() {
		done <- reader.Watch(ctx, policy.Settings{Scope: policy.System}, ".", func(change FileChange) error {
			if change.Refresh {
				notifications <- change
			}
			return nil
		})
	}()
	select {
	case <-notifications:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := os.WriteFile(filepath.Join(base, "changed.txt"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case change := <-notifications:
		if change.Rescan || !slices.Contains(change.Names, "changed.txt") {
			t.Fatalf("unscoped file change: %+v", change)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
