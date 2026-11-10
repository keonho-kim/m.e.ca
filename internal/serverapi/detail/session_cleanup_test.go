// Purpose: Keep unrelated sessions and cached documents responsive during a slow document read.
// Pattern: Controlled concurrent cleanup regression test.
// Usage: go test -race ./internal/serverapi/detail -run TestUnrelatedSessionCleanup.
// Related: internal/serverapi/detail/pool.go, internal/serverapi/detail/cache.go.
package detail

import (
	"testing"
	"testing/synctest"

	"github.com/keonho-kim/meca/internal/config/credentials"
)

func TestUnrelatedSessionCleanupDoesNotWaitForAnotherDocument(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := NewCache()
		defer cache.Close()
		vault := &credentials.Vault{}
		entered, release, readDone := make(chan struct{}), make(chan struct{}), make(chan error, 1)
		request := Request{Session: "session-a", Version: vault.RedactionVersion, NewMask: vault.NewChunkMask, LoadText: func(func(int) error) (string, error) { close(entered); <-release; return "document", nil }}
		go func() { _, _, _, err := cache.Read(t.Context(), request, 0, 64); readDone <- err }()
		<-entered
		closed := make(chan error, 1)
		go func() { closed <- cache.CloseSession("session-b") }()
		synctest.Wait()
		select {
		case err := <-closed:
			if err != nil {
				t.Error(err)
			}
		default:
			t.Error("unrelated session cleanup waited for the document reader")
		}
		close(release)
		if err := <-readDone; err != nil {
			t.Fatal(err)
		}
	})
}
