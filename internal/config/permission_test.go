// Purpose: Verify explicit permission names persist and invalid modes are rejected.
// Pattern: Configuration round-trip test.
// Usage: go test ./internal/config -run TestPermission.
// Related: internal/config/settings.go, internal/policy/permission.go.
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestPermissionRoundTrip(t *testing.T) {
	directory := t.TempDir()
	configuration := Defaults()
	configuration.Permission = policy.PermissionAllMighty
	if err := Save(directory, configuration); err != nil {
		t.Fatal(err)
	}
	got, err := Load(directory)
	if err != nil || got.Permission != policy.PermissionAllMighty {
		t.Fatalf("permission=%v error=%v", got.Permission, err)
	}
	for _, invalid := range []string{"permission='unknown'", "permission=3", "permission=''"} {
		if err := os.WriteFile(filepath.Join(directory, Filename), []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(directory); err == nil {
			t.Fatalf("accepted %s", invalid)
		}
	}
}
