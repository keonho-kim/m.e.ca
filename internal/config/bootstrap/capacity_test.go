// Purpose: Persist expanded compaction defaults in a fresh user's bootstrap configuration.
// Pattern: Bootstrap configuration round-trip test.
// Usage: go test ./internal/config/bootstrap -run TestCompactionCapacity.
// Related: internal/config/bootstrap/bootstrap.go, internal/agent/compact/usage.go.
package bootstrap

import (
	"github.com/keonho-kim/meca/internal/config"
	"path/filepath"
	"testing"
)

func TestCompactionCapacityDefaultsPersist(t *testing.T) {
	home := t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	settings, err := config.Load(filepath.Join(home, ".meca"))
	if err != nil {
		t.Fatal(err)
	}
	if settings.Compaction.MaxInputBytes != 16*1024*1024 || settings.Compaction.MaxSummaryBytes != 16*1024*1024 {
		t.Fatalf("defaults not persisted: %+v", settings.Compaction)
	}
}
