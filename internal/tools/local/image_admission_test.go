// Purpose: Keep image decoder admission shared and cancellable across local participants.
// Pattern: Deterministic resource ownership contract test.
// Usage: go test -race ./internal/tools/local -run TestSharedImageAdmission.
// Related: internal/tools/local/executor.go, internal/tools/local/skill_image.go.
package local

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"testing/synctest"
)

func TestSharedImageAdmission(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{}, 1)
		gate <- struct{}{}
		home := t.TempDir()
		first, err := NewExecutor(t.TempDir(), home, filepath.Join(home, ".meca"), func(v string) string { return v }, Options{Images: gate})
		if err != nil {
			t.Fatal(err)
		}
		defer first.Close()
		second, err := NewExecutor(t.TempDir(), home, filepath.Join(home, ".meca"), func(v string) string { return v }, Options{Images: gate})
		if err != nil {
			t.Fatal(err)
		}
		defer second.Close()
		if first.imageReads != second.imageReads {
			t.Fatal("participants did not share their decoder budget")
		}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := (&skillOperation{operation: operation{executor: second}}).readImage(ctx, nil)
			done <- err
		}()
		synctest.Wait()
		select {
		case err := <-done:
			t.Fatal("decoder bypassed the occupied budget", err)
		default:
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if len(gate) != 1 {
			t.Fatal("canceled waiter released another operation's permit")
		}
		<-gate
	})
}
