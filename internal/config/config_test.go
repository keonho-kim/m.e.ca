// Purpose: Verify private TOML configuration and reject unsupported settings.
// Pattern: Behavioral persistence tests.
// Usage: go test ./internal/config.
// Related: internal/config/config.go, internal/config/settings.go.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pelletier/go-toml/v2"
)

func TestLoadSaveWithDefaultPermission(t *testing.T) {
	directory := filepath.Join(t.TempDir(), ".meca")
	configuration, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(directory, configuration); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(configuration, restored); diff != "" {
		t.Fatalf("configuration round trip (-want +got):\n%s", diff)
	}
	body, err := os.ReadFile(filepath.Join(directory, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct{ Permission string }
	if err := toml.Unmarshal(body, &saved); err != nil || saved.Permission != "default" {
		t.Fatalf("default permission missing: %s %v", body, err)
	}
	info, err := os.Stat(filepath.Join(directory, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("configuration permissions: %v", info.Mode())
	}
}

func TestLoadRejectsInvalidSettingsWithoutChangingFile(t *testing.T) {
	for _, original := range []string{
		"[ui]\ninput_max_lines=20\n",
		"unexpected=true\n",
		"[compaction]\nunrecognized_option=16384\n",
	} {
		directory := t.TempDir()
		path := filepath.Join(directory, Filename)
		if err := os.WriteFile(path, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(directory); err == nil {
			t.Fatalf("accepted invalid configuration: %q", original)
		}
		if body, err := os.ReadFile(path); err != nil || string(body) != original {
			t.Fatalf("loading changed the existing configuration: %v", err)
		}
	}
}
