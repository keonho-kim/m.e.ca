// Purpose: Verify Keychain command cancellation, stdin secrecy and current OS-entry interoperability.
// Pattern: Controlled native command boundary tests without touching the user's Keychain.
// Usage: go test ./internal/config/credentials/osstore on macOS.
// Related: internal/config/credentials/osstore/store_darwin.go.
package osstore

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type keychainExit int

func (e keychainExit) Error() string { return "fixture exit" }
func (e keychainExit) ExitCode() int { return int(e) }

func TestKeychainUsesStdinAndRetainsCurrentEncoding(t *testing.T) {
	const value = "fixture-encryption-material"
	deletes := 0
	s := Store{Service: "meca", run: func(_ context.Context, input string, args ...string) ([]byte, error) {
		for _, arg := range args {
			if strings.Contains(arg, value) {
				t.Fatal("secret passed in process arguments")
			}
		}
		switch args[0] {
		case "-i":
			if !reflect.DeepEqual(args, []string{"-i"}) || !strings.Contains(input, keychainEncoding+base64.StdEncoding.EncodeToString([]byte(value))) {
				t.Fatal("wrong private stdin encoding")
			}
		case "find-generic-password":
			return []byte(keychainEncoding + base64.StdEncoding.EncodeToString([]byte(value)) + "\n"), nil
		case "delete-generic-password":
			if !reflect.DeepEqual(args, []string{"delete-generic-password", "-s", "meca"}) {
				t.Fatal("wrong service cleanup", args)
			}
			deletes++
			if deletes == 2 {
				return nil, keychainExit(44)
			}
		default:
			t.Fatal("unexpected command", args)
		}
		return nil, nil
	}}
	if err := s.Set(t.Context(), "encryption_fixture", value); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Get(t.Context(), "encryption_fixture"); err != nil || got != value {
		t.Fatal("existing encoding could not be read", err)
	}
	if err := s.Clear(t.Context()); err != nil || deletes != 2 {
		t.Fatal("cleanup did not stop at absence", err)
	}
	if err := s.Set(t.Context(), "invalid\nidentity", value); err == nil {
		t.Fatal("unsafe interactive command identity accepted")
	}
}

func TestKeychainNativeRoundTrip(t *testing.T) {
	if os.Getenv("MECA_TEST_KEYCHAIN") != "1" {
		t.Skip("requires explicit native Keychain verification")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	store := Store{Service: "meca-keychain-test-" + rand.Text()}
	defer func() {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(t.Context()), 10*time.Second)
		defer stop()
		if err := store.Clear(cleanup); err != nil {
			t.Errorf("remove disposable service %s: %v", store.Service, err)
		}
	}()
	const ref = "encryption_test"
	value := "fixture-" + rand.Text()
	if err := store.Set(ctx, ref, value); err != nil {
		t.Fatal("native write", err)
	}
	if got, err := store.Get(ctx, ref); err != nil || got != value {
		t.Fatal("native readback", err)
	}
	if err := store.Clear(ctx); err != nil {
		t.Fatal("native delete", err)
	}
	if _, err := store.Get(ctx, ref); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted native entry remains", err)
	}
}

func TestKeychainCancellationJoinsOwnedCall(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started, done := make(chan struct{}), make(chan error, 1)
	s := Store{Service: "meca", run: func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	go func() { _, err := s.Get(ctx, "encryption_fixture"); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
	s.run = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("canceled call reached OS command")
		return nil, nil
	}
	if _, err := s.Get(ctx, "encryption_fixture"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
