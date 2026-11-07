// Purpose: Verify timed re-encryption preserves secrets, backups and concurrent user changes.
// Pattern: Deterministic clock and channel-controlled persistence tests.
// Usage: go test -race ./internal/storage/credentialstore -run TestPeriodicRotation.
// Related: internal/storage/credentialstore/rotation.go, internal/storage/credentialstore/rotation_schedule.go.
package credentialstore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func currentCipher(t *testing.T, store *Store, ref string) (string, []byte) {
	t.Helper()
	var key string
	var body []byte
	if err := store.db.QueryRow("SELECT key_ref,sealed FROM credentials WHERE ref=?", ref).Scan(&key, &body); err != nil {
		t.Fatal(err)
	}
	return key, body
}

func TestPeriodicRotationSurvivesRestartAndPreservesBackup(t *testing.T) {
	s, keys, directory := credentialFixture(t)
	if err := s.Put(t.Context(), "provider", "scope", "unchanged-api-key"); err != nil {
		t.Fatal(err)
	}
	beforeKey, beforeCipher := currentCipher(t, s, "provider")
	backup, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 123)
	period := 24 * time.Hour
	if err := s.RotateDue(t.Context(), now, period); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), directory, keys, false)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err := reopened.RotateDue(t.Context(), now.Add(period-time.Nanosecond), period); err != nil {
		t.Fatal(err)
	}
	if got, _ := currentCipher(t, s, "provider"); got != beforeKey {
		t.Fatal("fresh or not-yet-due key was rotated")
	}
	if err := reopened.RotateDue(t.Context(), now.Add(period), period); err != nil {
		t.Fatal(err)
	}
	afterKey, afterCipher := currentCipher(t, s, "provider")
	if afterKey == beforeKey || bytes.Equal(beforeCipher, afterCipher) {
		t.Fatal("elapsed cryptoperiod did not replace encryption material")
	}
	if got, err := s.Get(t.Context(), "provider", "scope"); err != nil || got != "unchanged-api-key" {
		t.Fatal("rotation changed or lost the API key", err)
	}
	if err := reopened.RotateDue(t.Context(), now.Add(period+time.Hour), period); err != nil {
		t.Fatal(err)
	}
	if got, _ := currentCipher(t, s, "provider"); got != afterKey {
		t.Fatal("rotation repeated before the next period")
	}
	restored := t.TempDir()
	if err := os.WriteFile(filepath.Join(restored, Filename), backup, 0600); err != nil {
		t.Fatal(err)
	}
	old, err := Open(t.Context(), restored, keys, false)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	if got, err := old.Get(t.Context(), "provider", "scope"); err != nil || got != "unchanged-api-key" {
		t.Fatal("rotation invalidated an old backup", err)
	}
}

type rotationKeys struct {
	*testKeys
	fail, wait string
	started    chan struct{}
	release    chan struct{}
	once       sync.Once
}

func (k *rotationKeys) Get(ctx context.Context, ref string) (string, error) {
	if ref == k.fail {
		return "", errors.New("locked")
	}
	if ref == k.wait {
		k.once.Do(func() { close(k.started) })
		select {
		case <-k.release:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return k.testKeys.Get(ctx, ref)
}

func TestPeriodicRotationFailureDoesNotBlockOtherRecords(t *testing.T) {
	s, keys, _ := credentialFixture(t)
	for _, ref := range []string{"a", "b"} {
		if err := s.Put(t.Context(), ref, "scope", ref+"-secret"); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Unix(1_800_000_000, 0)
	if err := s.RotateDue(t.Context(), now, time.Hour); err != nil {
		t.Fatal(err)
	}
	a, sealed := currentCipher(t, s, "a")
	b, _ := currentCipher(t, s, "b")
	s.keys = &rotationKeys{testKeys: keys, fail: a}
	if err := s.RotateDue(t.Context(), now.Add(time.Hour), time.Hour); !errors.Is(err, ErrKeyUnavailable) {
		t.Fatal("missing OS key was not reported", err)
	}
	if got, raw := currentCipher(t, s, "a"); got != a || !bytes.Equal(raw, sealed) {
		t.Fatal("failed rotation changed the original ciphertext")
	}
	if got, _ := currentCipher(t, s, "b"); got == b {
		t.Fatal("one unavailable key blocked unrelated credentials")
	}
	s.keys = keys
	if err := s.RotateDue(t.Context(), now.Add(time.Hour), time.Hour); err != nil {
		t.Fatal(err)
	}
	if got, _ := currentCipher(t, s, "a"); got == a {
		t.Fatal("failed rotation was not retried")
	}
}

func TestPeriodicRotationCannotOverwriteUserSaveOrRunTwice(t *testing.T) {
	s, keys, directory := credentialFixture(t)
	if err := s.Put(t.Context(), "provider", "scope", "old-secret"); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	if err := s.RotateDue(t.Context(), now, time.Hour); err != nil {
		t.Fatal(err)
	}
	old, _ := currentCipher(t, s, "provider")
	waiting := &rotationKeys{testKeys: keys, wait: old, started: make(chan struct{}), release: make(chan struct{})}
	s.keys = waiting
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- s.RotateDue(ctx, now.Add(time.Hour), time.Hour) }()
	<-waiting.started
	other, err := Open(t.Context(), directory, keys, false)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if err := other.RotateDue(t.Context(), now.Add(time.Hour), time.Hour); err != nil {
		t.Fatal("another maintenance owner did not yield", err)
	}
	if err := other.Put(t.Context(), "provider", "scope", "user-replaced-secret"); err != nil {
		t.Fatal(err)
	}
	userKey, _ := currentCipher(t, other, "provider")
	close(waiting.release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if got, _ := currentCipher(t, other, "provider"); got != userKey {
		t.Fatal("maintenance overwrote a concurrent user save")
	}
	if got, err := other.Get(t.Context(), "provider", "scope"); err != nil || got != "user-replaced-secret" {
		t.Fatal("new API key became unreadable", err)
	}
}
