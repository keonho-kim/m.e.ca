// Purpose: Cancel OS-key operations without losing an existing encrypted credential or starting canceled I/O.
// Pattern: Channel-controlled credential boundary regression tests.
// Usage: go test -race ./internal/config/credentials -run TestVault.
// Related: internal/config/credentials/keyring.go, internal/storage/credentialstore/crypto.go.
package credentials

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/config/credentials/credentialtest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type waitingKeys struct {
	*credentialtest.Memory
	started chan struct{}
	write   bool
}

func (k *waitingKeys) Set(ctx context.Context, ref, value string) error {
	if !k.write {
		return k.Memory.Set(ctx, ref, value)
	}
	close(k.started)
	<-ctx.Done()
	return ctx.Err()
}
func (k *waitingKeys) Get(ctx context.Context, ref string) (string, error) {
	if k.write {
		return k.Memory.Get(ctx, ref)
	}
	close(k.started)
	<-ctx.Done()
	return "", ctx.Err()
}

func TestVaultCancellationPreservesEncryptedCredential(t *testing.T) {
	for _, write := range []bool{false, true} {
		keys := &credentialtest.Memory{}
		v := &Vault{State: t.TempDir(), Keys: keys}
		if err := v.Set(t.Context(), "provider", "scope", "old-secret"); err != nil {
			t.Fatal(err)
		}
		waiting := &waitingKeys{Memory: keys, started: make(chan struct{}), write: write}
		v.Keys = waiting
		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan error, 1)
		go func() {
			if write {
				result <- v.Set(ctx, "provider", "scope", "new-secret")
			} else {
				_, err := v.Get(ctx, "provider", "scope")
				result <- err
			}
		}()
		<-waiting.started
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatal("OS cancellation lost", err)
		}
		v.Keys = keys
		if value, err := v.Get(t.Context(), "provider", "scope"); err != nil || value != "old-secret" {
			t.Fatal("cancellation lost the active credential", err)
		}
	}
}

func TestVaultCanceledBeforeIOAndNormalizesHTTPWhitespace(t *testing.T) {
	v := &Vault{State: filepath.Join(t.TempDir(), "not-created"), Keys: &credentialtest.Memory{}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := v.Set(ctx, "provider", "scope", "secret"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := os.Stat(v.State); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("canceled operation created state", err)
	}
	if err := v.Set(t.Context(), "provider", "scope", " \tprovider-secret\t "); err != nil {
		t.Fatal(err)
	}
	if value, err := v.Get(t.Context(), "provider", "scope"); err != nil || value != "provider-secret" {
		t.Fatal("HTTP whitespace normalization differs", err)
	}
	if strings.Contains(v.Redact("server rejected provider-secret"), "provider-secret") {
		t.Fatal("wire credential was not remembered")
	}
}
