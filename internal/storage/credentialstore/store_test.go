// Purpose: Verify authenticated encryption, durable rotation and fail-closed recovery behavior.
// Pattern: SQLite and controllable key-store contract tests.
// Usage: go test -race ./internal/storage/credentialstore.
// Related: internal/storage/credentialstore/crypto.go, internal/storage/credentialstore/database.go.
package credentialstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type testKeys struct {
	mu               sync.Mutex
	values           map[string]string
	failSet, failGet bool
	writes           int
}

func (k *testKeys) Set(_ context.Context, ref, value string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.failSet {
		return errors.New("locked")
	}
	if k.values == nil {
		k.values = map[string]string{}
	}
	k.values[ref] = value
	k.writes++
	return nil
}
func (k *testKeys) Get(_ context.Context, ref string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.failGet {
		return "", errors.New("locked")
	}
	v, ok := k.values[ref]
	if !ok {
		return "", errors.New("missing")
	}
	return v, nil
}
func credentialFixture(t *testing.T) (*Store, *testKeys, string) {
	t.Helper()
	directory := t.TempDir()
	keys := &testKeys{}
	s, err := Open(t.Context(), directory, keys, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, keys, directory
}

func TestRotationRetainsOldBackupsAndUsesIndependentKeys(t *testing.T) {
	s, keys, directory := credentialFixture(t)
	const ref, scope = "provider", "provider-scope"
	if err := s.Put(t.Context(), ref, scope, "original-secret"); err != nil {
		t.Fatal(err)
	}
	var oldRef string
	var oldSealed []byte
	if err := s.db.QueryRow("SELECT key_ref,sealed FROM credentials WHERE ref=?", ref).Scan(&oldRef, &oldSealed); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(backup, []byte("original-secret")) {
		t.Fatal("plaintext credential in SQLite")
	}
	if err := s.Put(t.Context(), ref, scope, "replacement-secret"); err != nil {
		t.Fatal(err)
	}
	var nextRef string
	var nextSealed []byte
	if err := s.db.QueryRow("SELECT key_ref,sealed FROM credentials WHERE ref=?", ref).Scan(&nextRef, &nextSealed); err != nil {
		t.Fatal(err)
	}
	if oldRef == nextRef || bytes.Equal(oldSealed, nextSealed) {
		t.Fatal("credential version reused encryption material")
	}
	if _, err := keys.Get(t.Context(), oldRef); err != nil {
		t.Fatal("rotation deleted an older decryption key")
	}
	restored := t.TempDir()
	if err := os.WriteFile(filepath.Join(restored, Filename), backup, 0600); err != nil {
		t.Fatal(err)
	}
	previous, err := Open(t.Context(), restored, keys, false)
	if err != nil {
		t.Fatal(err)
	}
	defer previous.Close()
	if value, err := previous.Get(t.Context(), ref, scope); err != nil || value != "original-secret" {
		t.Fatal("older backup lost after rotation", err)
	}
	if value, err := s.Get(t.Context(), ref, scope); err != nil || value != "replacement-secret" {
		t.Fatal("current credential lost", err)
	}
}

func TestAuthenticationBindsReferenceProviderAndCiphertext(t *testing.T) {
	s, _, _ := credentialFixture(t)
	if err := s.Put(t.Context(), "provider", "original-connection", "secret"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(t.Context(), "provider", "different-connection"); !errors.Is(err, ErrIntegrity) {
		t.Fatal("credential crossed its provider boundary", err)
	}
	if err := s.Put(t.Context(), "provider", "different-connection", "another-secret"); !errors.Is(err, ErrIntegrity) {
		t.Fatal("credential reference moved to another provider", err)
	}
	if value, err := s.Get(t.Context(), "provider", "original-connection"); err != nil || value != "secret" {
		t.Fatal("rejected scope update lost the original key", err)
	}
	var sealed []byte
	if err := s.db.QueryRow("SELECT sealed FROM credentials WHERE ref='provider'").Scan(&sealed); err != nil {
		t.Fatal(err)
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := s.db.Exec("UPDATE credentials SET sealed=? WHERE ref='provider'", sealed); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(t.Context(), "provider", "original-connection"); !errors.Is(err, ErrIntegrity) {
		t.Fatal("tampered ciphertext accepted", err)
	}
	if _, err := s.db.Exec("UPDATE credentials SET ref='another',scope='different-connection' WHERE ref='provider'"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(t.Context(), "another", "different-connection"); !errors.Is(err, ErrIntegrity) {
		t.Fatal("record identity mutation accepted", err)
	}
}

func TestMissingOSKeyNeverRecreatesOrOverwritesData(t *testing.T) {
	s, keys, directory := credentialFixture(t)
	if err := s.Put(t.Context(), "provider", "scope", "secret"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil {
		t.Fatal(err)
	}
	writes := keys.writes
	keys.failGet = true
	if _, err := s.Get(t.Context(), "provider", "scope"); !errors.Is(err, ErrKeyUnavailable) {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil || !bytes.Equal(before, after) || keys.writes != writes {
		t.Fatal("missing OS key caused data replacement", err)
	}
	keys.failGet = false
	if value, err := s.Get(t.Context(), "provider", "scope"); err != nil || value != "secret" {
		t.Fatal("restored OS key did not recover ciphertext", err)
	}
}

func TestFailedOSWriteOrReadbackPreservesPreviousCredential(t *testing.T) {
	s, keys, _ := credentialFixture(t)
	if err := s.Put(t.Context(), "provider", "scope", "old-secret"); err != nil {
		t.Fatal(err)
	}
	for _, verify := range []bool{false, true} {
		keys.failSet, keys.failGet = !verify, verify
		if err := s.Put(t.Context(), "provider", "scope", "new-secret"); !errors.Is(err, ErrKeyUnavailable) {
			t.Fatal(err)
		}
		keys.failSet, keys.failGet = false, false
		if value, err := s.Get(t.Context(), "provider", "scope"); err != nil || value != "old-secret" {
			t.Fatal("failed OS operation replaced the active credential", err)
		}
	}
}

func TestFailedSQLWritePreservesPreviousCredential(t *testing.T) {
	s, keys, directory := credentialFixture(t)
	if err := s.Put(t.Context(), "provider", "scope", "old-secret"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(t.Context(), "provider", "scope", "new-secret"); err == nil {
		t.Fatal("closed database accepted write")
	}
	reopened, err := Open(t.Context(), directory, keys, false)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if value, err := reopened.Get(t.Context(), "provider", "scope"); err != nil || value != "old-secret" {
		t.Fatal("failed SQL write lost the previous credential", err)
	}
}

func TestUnsupportedDatabaseIsPreserved(t *testing.T) {
	s, keys, directory := credentialFixture(t)
	if _, err := s.db.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open(t.Context(), directory, keys, true); err == nil {
		t.Fatal("unsupported database was recreated")
	}
	after, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("unsupported database was changed", err)
	}
}

func TestConcurrentHandlesDoNotLoseEncryptionKeys(t *testing.T) {
	_, keys, directory := credentialFixture(t)
	var writers sync.WaitGroup
	for i := range 6 {
		writers.Go(func() {
			s, err := Open(t.Context(), directory, keys, true)
			if err != nil {
				t.Error(err)
				return
			}
			defer s.Close()
			secret := fmt.Sprintf("credential-%d", i)
			if err := s.Put(t.Context(), "provider", "scope", secret); err != nil {
				t.Error(err)
			}
		})
	}
	writers.Wait()
	s, err := Open(t.Context(), directory, keys, false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	value, err := s.Get(t.Context(), "provider", "scope")
	if err != nil || !strings.HasPrefix(value, "credential-") {
		t.Fatal("concurrent encrypted update became unreadable", err)
	}
	if keys.writes != 6 {
		t.Fatal("independent key versions were not retained", keys.writes)
	}
}
