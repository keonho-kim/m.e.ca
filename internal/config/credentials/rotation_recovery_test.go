// Purpose: Keep credential maintenance retrying even when central diagnostic output fails.
// Pattern: Fake-clock fault recovery regression test.
// Usage: go test ./internal/config/credentials -run TestRotationWorkerRecoversDiagnosticFailure.
// Related: internal/config/credentials/rotation.go, internal/config/credentials/rotation_test.go.
package credentials

import (
	"errors"
	"github.com/keonho-kim/meca/internal/diagnostics"
	"testing"
	"testing/synctest"
	"time"
)

type unavailableLog struct{}

func (unavailableLog) Write([]byte) (int, error) { return 0, errors.New("private output failure") }

func TestRotationWorkerRecoversDiagnosticFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keys := &rotatingKeys{}
		v := &Vault{State: t.TempDir(), Keys: keys}
		if err := v.Set(t.Context(), "provider", "scope", "fixture-key"); err != nil {
			t.Fatal(err)
		}
		sink, err := diagnostics.New(diagnostics.Options{Writer: unavailableLog{}, Redact: v.Redact})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = sink.Close() }()
		stop := v.StartRotation(diagnostics.WithSink(t.Context(), sink))
		defer func() { _ = stop() }()
		synctest.Wait()
		keys.failed.Store(true)
		time.Sleep(rotationPeriod)
		synctest.Wait()
		if sink.Healthy() {
			t.Fatal("diagnostic output did not fail")
		}
		keys.failed.Store(false)
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
