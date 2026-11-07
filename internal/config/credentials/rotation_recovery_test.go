// Purpose: Keep maintenance retrying after its diagnostic storage temporarily fails.
// Pattern: Fake-clock fault recovery regression test.
// Usage: go test ./internal/config/credentials -run TestRotationWorkerRecoversDiagnosticFailure.
// Related: internal/config/credentials/rotation.go, internal/config/credentials/rotation_test.go.
package credentials

import (
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"
)

func TestRotationWorkerRecoversDiagnosticFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keys := &rotatingKeys{}
		v := &Vault{State: t.TempDir(), Keys: keys}
		if err := v.Set(t.Context(), "provider", "scope", "fixture-key"); err != nil {
			t.Fatal(err)
		}
		stop := v.StartRotation(t.Context())
		defer func() { _ = stop() }()
		synctest.Wait()
		logs := filepath.Join(v.State, "logs")
		if err := os.WriteFile(logs, []byte("blocked"), 0600); err != nil {
			t.Fatal(err)
		}
		keys.failed.Store(true)
		time.Sleep(rotationPeriod)
		synctest.Wait()
		keys.failed.Store(false)
		if err := os.Remove(logs); err != nil {
			t.Fatal(err)
		}
		time.Sleep(rotationPoll)
		synctest.Wait()
		if keys.writes.Load() != 2 {
			t.Fatal("recovered storage did not resume maintenance", stop())
		}
		if err := stop(); err != nil {
			t.Fatal("recovered diagnostic error remained terminal", err)
		}
	})
}
