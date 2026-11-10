// Purpose: Reject foreign conversation identities before querying a store bound to one session.
// Pattern: Persistence ownership regression test.
// Usage: go test ./internal/storage/sqlite -run TestStoreRejectsForeignSession.
// Related: internal/storage/sqlite/session.go, internal/storage/sqlite/commit.go.
package sqlite

import "testing"

func TestStoreRejectsForeignSession(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, read := range []func() error{
		func() error { _, err := s.Selection(t.Context(), "foreign"); return err },
		func() error { _, err := s.FileSelections(t.Context(), "foreign"); return err },
		func() error { _, err := s.Memory(t.Context(), "foreign"); return err },
		func() error { _, _, err := s.Timeline(t.Context(), "foreign", 0, 5); return err },
	} {
		if err := read(); err == nil {
			t.Fatal("foreign session accepted")
		}
	}
}
