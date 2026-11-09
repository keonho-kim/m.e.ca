// Purpose: Bound catalog and opaque path lifetimes without permanently consuming reader capacity.
// Pattern: Deterministic cache lifetime behavior tests.
// Usage: go test -race ./internal/skills -run 'TestCatalogRetention|TestPathRegistry'.
// Related: internal/skills/catalog_retention.go, internal/skills/path_ids.go.
package skills

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestCatalogRetentionExpiresAndEvictsIndependentSnapshots(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r := New(t.TempDir(), t.TempDir(), Options{Now: func() time.Time { return now }})
	for i := range catalogSnapshots + 1 {
		s := &catalogSnapshot{id: fmt.Sprint(i), at: now, items: []catalogEntry{newCatalogEntry(instruction.Metadata{Name: fmt.Sprint(i)})}}
		if _, err := r.retainCatalog(s); err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Second)
	}
	if len(r.snapshots) != catalogSnapshots {
		t.Fatal(len(r.snapshots))
	}
	if _, err := r.pageSnapshot(t.Context(), "0"); !errors.Is(err, ErrInvalid) {
		t.Fatal("old snapshot not evicted", err)
	}
	if _, err := r.pageSnapshot(t.Context(), "1"); err != nil {
		t.Fatal("live independent traversal lost", err)
	}
	now = now.Add(catalogLifetime + time.Second)
	if _, err := r.pageSnapshot(t.Context(), "1"); !errors.Is(err, ErrInvalid) {
		t.Fatal("expired cursor accepted", err)
	}
	if _, err := r.retainCatalog(&catalogSnapshot{id: "large", at: now, items: []catalogEntry{newCatalogEntry(instruction.Metadata{Description: strings.Repeat("x", catalogBytes)})}}); !errors.Is(err, ErrLimit) {
		t.Fatal("oversized snapshot retained", err)
	}
}

func TestPathRegistryReusesLeastRecentlyUsedSlots(t *testing.T) {
	r := New(t.TempDir(), t.TempDir(), Options{})
	for i := range 128 {
		r.rememberPath(fmt.Sprint(i), fmt.Sprintf("/skills/%d", i))
	}
	if r.pathFor("0") == "" {
		t.Fatal("missing path")
	}
	r.rememberPath("new", "/skills/new")
	if r.pathFor("0") == "" || r.pathFor("1") != "" || len(r.paths) != 128 {
		t.Fatal("path registry did not reuse the oldest slot")
	}
	r.rememberPath("1", "/skills/1")
	if r.pathFor("1") == "" {
		t.Fatal("evicted path could not be reloaded")
	}
}
