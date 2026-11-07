//go:build linux

// Purpose: Check real Secret Service storage with a disposable service name in an isolated test process.
// Pattern: Opt-in credential adapter integration test.
// Usage: MECA_TEST_SECRET_SERVICE=1 GOFLAGS='-tags=godbus_no_tcp' go test ./internal/config/credentials -run TestCredentialSecretService.
// Related: internal/config/credentials/dbus_linux_test.go, internal/config/credentials/keyring.go, docs/development.md.
package credentials

import (
	"crypto/rand"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/config/credentials/osstore"
)

func TestCredentialSecretService(t *testing.T) {
	if os.Getenv("MECA_TEST_SECRET_SERVICE") != "1" {
		t.Skip("requires an isolated session bus with an unlocked Secret Service")
	}
	address := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if !strings.HasPrefix(address, "unix:") {
		t.Fatal("the integration fixture requires an explicit Unix session bus")
	}
	if err := credentialDBusProcess(t, address, "secret-service"); err != nil {
		t.Fatal(err)
	}
}

func credentialSecretServiceRoundTrip(t *testing.T) {
	t.Helper()
	name := "meca-credential-test-" + rand.Text()
	store := osstore.Store{Service: name}
	const user, secret = "fixture-user", "fixture-secret-not-an-api-key"
	if err := store.Set(t.Context(), user, secret); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := store.Clear(t.Context()); err != nil && !errors.Is(err, osstore.ErrNotFound) {
			t.Error("remove test service:", err)
		}
	}()
	value, err := store.Get(t.Context(), user)
	if err != nil || value != secret {
		t.Fatalf("secret retrieval mismatch: %v", err)
	}
	if err := store.Clear(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(t.Context(), user); !errors.Is(err, osstore.ErrNotFound) {
		t.Fatalf("deleted secret remains: %v", err)
	}
}
