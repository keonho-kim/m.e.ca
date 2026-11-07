// Purpose: Verify automatic rotation timing, private failures and joined shutdown without real waits.
// Pattern: Fake-clock worker and channel-controlled OS-key tests.
// Usage: go test -race ./internal/config/credentials -run TestRotationWorker.
// Related: internal/config/credentials/rotation.go, internal/storage/credentialstore/rotation.go.
package credentials

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/keonho-kim/meca/internal/config/credentials/credentialtest"
)

type rotatingKeys struct {
	credentialtest.Memory
	writes  atomic.Int64
	failed  atomic.Bool
	started chan struct{}
	once    sync.Once
}

func (k *rotatingKeys) Set(ctx context.Context, ref, value string) error {
	if err := k.Memory.Set(ctx, ref, value); err != nil {
		return err
	}
	k.writes.Add(1)
	return nil
}

func (k *rotatingKeys) Get(ctx context.Context, ref string) (string, error) {
	if k.started != nil {
		k.once.Do(func() { close(k.started) })
		<-ctx.Done()
		return "", ctx.Err()
	}
	if k.failed.Load() {
		return "", fmt.Errorf("untrusted-backend-private-fixture")
	}
	return k.Memory.Get(ctx, ref)
}

func TestRotationWorkerPeriodAndShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keys := &rotatingKeys{}
		v := &Vault{State: t.TempDir(), Keys: keys}
		if err := v.Set(t.Context(), "provider", "scope", "provider-secret"); err != nil {
			t.Fatal(err)
		}
		stop := v.StartRotation(t.Context())
		defer func() {
			if err := stop(); err != nil {
				t.Error(err)
			}
		}()
		synctest.Wait()
		time.Sleep(rotationPeriod - rotationPoll)
		synctest.Wait()
		if keys.writes.Load() != 1 {
			t.Fatal("not-yet-due credential rotated")
		}
		time.Sleep(rotationPoll)
		synctest.Wait()
		if keys.writes.Load() != 2 {
			t.Fatal("hourly worker did not rotate after the internal period")
		}
		if got, err := v.Get(t.Context(), "provider", "scope"); err != nil || got != "provider-secret" {
			t.Fatal("automatic rotation changed the API key", err)
		}
		if err := stop(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(rotationPeriod)
		synctest.Wait()
		if keys.writes.Load() != 2 {
			t.Fatal("stopped maintenance performed more work")
		}
	})
}

func TestRotationWorkerFailureIsPrivateAndRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keys := &rotatingKeys{}
		v := &Vault{State: t.TempDir(), Keys: keys}
		if err := v.Set(t.Context(), "provider", "scope", "provider-secret"); err != nil {
			t.Fatal(err)
		}
		stop := v.StartRotation(t.Context())
		defer func() {
			if err := stop(); err != nil {
				t.Error(err)
			}
		}()
		synctest.Wait()
		keys.failed.Store(true)
		time.Sleep(rotationPeriod)
		synctest.Wait()
		body, err := os.ReadFile(filepath.Join(v.State, "logs", "credential-rotation.log"))
		if err != nil || !strings.Contains(string(body), "deferred") || strings.Contains(string(body), "provider-secret") || strings.Contains(string(body), "untrusted-backend") {
			t.Fatal("failure was missing or disclosed sensitive backend text", err)
		}
		if keys.writes.Load() != 1 {
			t.Fatal("unavailable old key caused a replacement")
		}
		keys.failed.Store(false)
		time.Sleep(rotationPoll)
		synctest.Wait()
		if keys.writes.Load() != 2 {
			t.Fatal("deferred rotation was not retried")
		}
	})
}

func TestRotationWorkerJoinsCanceledOSAccess(t *testing.T) {
	keys := &rotatingKeys{}
	v := &Vault{State: t.TempDir(), Keys: keys}
	if err := v.Set(t.Context(), "provider", "scope", "provider-secret"); err != nil {
		t.Fatal(err)
	}
	if err := v.rotateOnce(t.Context(), time.Now().Add(-rotationPeriod-time.Hour)); err != nil {
		t.Fatal(err)
	}
	keys.started = make(chan struct{})
	stop := v.StartRotation(t.Context())
	<-keys.started
	if err := stop(); err != nil {
		t.Fatal(err)
	}
	keys.started = nil // Stop joined every callback before returning.
	if got, err := v.Get(t.Context(), "provider", "scope"); err != nil || got != "provider-secret" {
		t.Fatal("canceling maintenance lost the credential", err)
	}
}

func TestRotationWorkerDoesNotInitializeAnUnusedVault(t *testing.T) {
	state := filepath.Join(t.TempDir(), "not-created")
	v := &Vault{State: state, Keys: &rotatingKeys{}}
	if err := v.rotateOnce(t.Context(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("maintenance created unused credential state", err)
	}
}
