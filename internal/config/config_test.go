// Purpose: Verify private TOML configuration and project-setting precedence.
// Pattern: Behavioral persistence tests.
// Usage: go test ./internal/config.
// Related: internal/config/config.go, internal/policy/policy.go.

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/keonho-kim/v-code/internal/policy"
)

func TestLoadSaveAndEffectiveSettings(t *testing.T) {
	directory := filepath.Join(t.TempDir(), ".v")
	configuration, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Defaults != (policy.Settings{Permission: policy.AllowAll, Scope: policy.System}) {
		t.Fatalf("unsafe defaults: %+v", configuration.Defaults)
	}
	configuration.Defaults.Permission = policy.AllowAll
	if err := Save(directory, configuration); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(directory)
	if err != nil || !reflect.DeepEqual(restored, configuration) {
		t.Fatalf("restore: %+v, %v", restored, err)
	}
	actual, err := Effective(restored.Defaults, policy.Settings{Scope: policy.Directory})
	if err != nil || actual.Permission != policy.AllowAll || actual.Scope != policy.System {
		t.Fatalf("effective settings: %+v, %v", actual, err)
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
		"[defaults]\npermission='anything'\n",
		"[ui]\ninput_max_lines=20\n",
		"[defaults]\nscope='outside'\n",
		"[defaults]\nscope='directory'\n",
		"unexpected=true\n",
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
	if configuration.Compaction.ThresholdTokens != 200000 {
		t.Fatal("wrong default token threshold")
	}
	configuration.Compaction.ThresholdTokens = 128000
	if err := Save(directory, configuration); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(directory)
	if err != nil || loaded.Compaction.ThresholdTokens != 128000 {
		t.Fatalf("threshold override: %+v %v", loaded, err)
	}
}
