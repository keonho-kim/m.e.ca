// Purpose: Verify private TOML configuration and reject unsupported settings.
// Pattern: Behavioral persistence tests.
// Usage: go test ./internal/config.
// Related: internal/config/config.go, internal/config/settings.go.

package config

import (
	"github.com/google/go-cmp/cmp"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err != nil || !strings.Contains(string(body), "permission = 'default'") || strings.Contains(string(body), "[defaults]") {
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
		"[[model_profiles]]\nprovider=\"vllm\"\nmodel=\"alias\"\nprofile=\"generic\"\n",
		"[session]\nrestore_last=true\n",
		"[ui]\ntheme='classic'\n",
		"[compaction]\ntarget_input_tokens=16384\n",
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

func TestCompactionThresholdIsConfigurable(t *testing.T) {
	directory := t.TempDir()
	configuration := Defaults()
	if configuration.Compaction.ReserveTokens != 16384 {
		t.Fatal("wrong default token threshold")
	}
	configuration.Compaction.ReserveTokens = 128000
	if err := Save(directory, configuration); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(directory)
	if err != nil || loaded.Compaction.ReserveTokens != 128000 {
		t.Fatalf("threshold override: %+v %v", loaded, err)
	}
}
