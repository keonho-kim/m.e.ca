// Purpose: Preserve current credentials when the durable OS-key allocation ceiling is exhausted.
// Pattern: Capacity failure injection at the publication boundary.
// Usage: go test ./internal/storage/credentialstore -run TestKeyCapacity.
// Related: internal/storage/credentialstore/capacity.go, internal/storage/credentialstore/crypto.go.
package credentialstore

import (
	"errors"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/storage/capacity"
)

func TestKeyCapacityPreservesCurrentValue(t *testing.T) {
	s, keys, directory := credentialFixture(t)
	if err := s.Put(t.Context(), "provider", "scope", "original"); err != nil {
		t.Fatal(err)
	}
	before := keys.writes
	if err := statefs.Write(filepath.Join(directory, "credential-key-capacity"), "allocated", []byte("v1\n"+strconv.Itoa(maxKeyVersions)+"\n")); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(t.Context(), "provider", "scope", "replacement"); !errors.Is(err, capacity.ErrLimit) {
		t.Fatal(err)
	}
	if keys.writes != before {
		t.Fatal("exhausted budget published an OS key")
	}
	if got, err := s.Get(t.Context(), "provider", "scope"); err != nil || got != "original" {
		t.Fatal("current credential lost", err)
	}
}
