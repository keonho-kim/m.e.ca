// Purpose: Avoid publishing OS keys for definitively rejected credential writes.
// Pattern: Key-store boundary regression tests.
// Usage: go test -race ./internal/storage/credentialstore -run TestRejectedWrite.
// Related: internal/storage/credentialstore/store.go, internal/storage/credentialstore/capacity.go.
package credentialstore

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/keonho-kim/meca/internal/storage/capacity"
)

func TestRejectedWriteDoesNotPublishOSKey(t *testing.T) {
	for _, full := range []bool{false, true} {
		t.Run(fmt.Sprint(full), func(t *testing.T) {
			s, keys, _ := credentialFixture(t)
			if err := s.Put(t.Context(), "provider", "scope", "initial-fixture"); err != nil {
				t.Fatal(err)
			}
			ref, scope, expected := "provider", "wrong-scope", ErrIntegrity
			if full {
				for i := range maxCredentials - 1 {
					_, err := s.db.Exec("INSERT INTO credentials(ref,scope,key_ref,sealed) VALUES(?,?,?,?)", fmt.Sprint(i), "scope", "encryption_abcdefghijklmnopqrstuvwxyz", make([]byte, 29))
					if err != nil {
						t.Fatal(err)
					}
				}
				ref, scope, expected = "new-provider", "scope", capacity.ErrLimit
			}
			before := keys.writes
			if err := s.Put(t.Context(), ref, scope, "replacement-fixture"); !errors.Is(err, expected) {
				t.Fatal(err)
			}
			if keys.writes != before {
				t.Fatal("rejected write published an unused OS key")
			}
		})
	}
}

func TestRejectedConcurrentWritesDoNotConsumeExtraKeys(t *testing.T) {
	s, keys, directory := credentialFixture(t)
	for i := range maxCredentials - 1 {
		_, err := s.db.Exec("INSERT INTO credentials(ref,scope,key_ref,sealed) VALUES(?,?,?,?)", fmt.Sprint(i), "scope", "encryption_abcdefghijklmnopqrstuvwxyz", make([]byte, 29))
		if err != nil {
			t.Fatal(err)
		}
	}
	var writers sync.WaitGroup
	for i := range 8 {
		writers.Go(func() {
			other, err := Open(t.Context(), directory, keys, false)
			if err != nil {
				t.Error(err)
				return
			}
			defer other.Close()
			if err := other.Put(t.Context(), fmt.Sprintf("new-%d", i), "scope", "fixture"); err != nil && !errors.Is(err, capacity.ErrLimit) {
				t.Error(err)
			}
		})
	}
	writers.Wait()
	if keys.writes != 1 {
		t.Fatal("rejected concurrent writes published unused keys", keys.writes)
	}
}
