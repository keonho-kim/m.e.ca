// Purpose: Verify private TOML configuration and reject unsupported settings.
// Pattern: Behavioral persistence tests.
// Usage: go test ./internal/config.
// Related: internal/config/config.go, internal/config/settings.go.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	if err != nil || !strings.Contains(string(body), "permission = 'default'") {
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

func TestLoadRejectsInvalidAndUnknownSettings(t *testing.T) {
	for _, content := range []string{
		"[ui]\ninput_max_lines=20\n",
		"unexpected=true\n",
	} {
		directory := t.TempDir()
		if err := os.WriteFile(filepath.Join(directory, "config.toml"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(directory); err == nil {
			t.Errorf("accepted %q", content)
		}
	}
}

func TestLoadReportsUnsupportedSettingWithoutChangingFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, Filename)
	original := "[compaction]\nreserve_tokens = 16384\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(directory); err == nil || !strings.Contains(err.Error(), "remove [compaction].reserve_tokens") {
		t.Fatalf("missing repair guidance for the unsupported setting: %v", err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != original {
		t.Fatalf("loading changed the existing configuration: %v", err)
	}
}
