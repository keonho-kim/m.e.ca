// Purpose: Verify two terminal processes cannot own the same resumed session concurrently.
// Pattern: Repository lifecycle tests.
// Usage: go test ./internal/storage/sqlite -run TestSessionLease.
// Related: internal/storage/sqlite/locks_unix.go.
package sqlite

import (
	"testing"
)

func TestSessionLease(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	lease, err := store.LeaseSession()
	if err != nil {
		t.Fatal(err)
	}
	if second, err := store.LeaseSession(); err == nil {
		if closeErr := second.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("second owner acquired active session")
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	lease, err = store.LeaseSession()
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
}
