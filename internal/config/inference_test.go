// Purpose: Verify role defaults are stored and zero-valued temperature overrides survive loading.
// Pattern: Configuration behavior tests.
// Usage: go test ./internal/config -run TestInference.
// Related: internal/config/config.go, internal/agent/inference/settings.go.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent/inference"
)

func TestAgentConfigurationUsesTopLevelTable(t *testing.T) {
	directory := t.TempDir()
	if err := Save(directory, Defaults()); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil || !strings.Contains(string(body), "[agent]") {
		t.Fatalf("main-agent configuration must use [agent]: %s %v", body, err)
	}
	if err := os.WriteFile(filepath.Join(directory, Filename), []byte("[agent]\ntemperature=0.0\nmax_output_tokens=12345\n"), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(directory)
	if err != nil || loaded.Agent.Temperature != 0 || loaded.Agent.MaxOutputTokens != 12345 || loaded.InferenceSettings().Agent != loaded.Agent {
		t.Fatalf("top-level agent settings were lost: %+v %v", loaded.Agent, err)
	}
}

func TestSupportingRolesUseTopLevelTables(t *testing.T) {
	directory := t.TempDir()
	if err := Save(directory, Defaults()); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{"[agent]", "[compaction]", "[diagnostics]"} {
		if strings.Count(string(body), section) != 1 {
			t.Fatalf("expected one %s section: %s", section, body)
		}
	}
	content := "[compaction]\ntemperature=0.0\nmax_output_tokens=4321\nreserve_tokens=123456\nkeep_recent_tokens=2048\nmax_input_bytes=1048576\nmax_summary_bytes=1048576\n\n[diagnostics]\ntemperature=0.0\nmax_output_tokens=2345\n"
	if err := os.WriteFile(filepath.Join(directory, Filename), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	parameters := loaded.InferenceSettings()
	if parameters.Compaction.Temperature != 0 || parameters.Compaction.MaxOutputTokens != 4321 || parameters.Compaction.Thinking != "off_if_supported" || parameters.Diagnostic.Temperature != 0 || parameters.Diagnostic.MaxOutputTokens != 2345 {
		t.Fatalf("top-level role overrides lost: %+v", parameters)
	}
	if loaded.Compaction.ReserveTokens != 123456 || loaded.Compaction.KeepRecentTokens != 2048 || loaded.Compaction.MaxInputBytes != 1048576 || loaded.Compaction.MaxSummaryBytes != 1048576 {
		t.Fatalf("compaction policy lost: %+v", loaded.Compaction)
	}
	if err := Save(directory, loaded); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(directory)
	if err != nil || restored.InferenceSettings() != parameters || restored.Compaction != loaded.Compaction {
		t.Fatalf("role/policy round trip lost settings: %+v %v", restored, err)
	}
}

func TestInferenceDefaultsAndOverrides(t *testing.T) {
	directory := t.TempDir()
	if err := Save(directory, Defaults()); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"agent", "compaction", "diagnostics"} {
		if !strings.Contains(string(body), "["+role+"]") {
			t.Fatalf("missing persisted role %s", role)
		}
	}
	loaded, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	settings := loaded.InferenceSettings()
	for _, role := range []inference.Role{inference.Agent, inference.Compaction, inference.Diagnostic} {
		parameters, err := inference.Resolve(&settings, role)
		want := 12800
		if role == inference.Agent {
			want = 0
		}
		if role == inference.Compaction {
			want = 8192
		}
		if err != nil || parameters.MaxOutputTokens != want {
			t.Fatalf("persisted %s default=%+v, want %d: %v", role, parameters, want, err)
		}
		if (role == inference.Agent) && parameters.Temperature != 1 {
			t.Fatalf("persisted %s temperature=%v; want 1", role, parameters.Temperature)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, Filename), []byte("[compaction]\ntemperature=0.0\nmax_output_tokens=1536\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	if c.Compaction.Temperature != 0 || c.Compaction.MaxOutputTokens != 1536 || c.Agent.Temperature != 1 {
		t.Fatalf("role override lost: %+v", c.InferenceSettings())
	}
}
func TestInferenceRejectsInvalidSettings(t *testing.T) {
	for _, value := range []string{"temperature=-1", "temperature=nan", "temperature=inf", "max_output_tokens=-1", "max_output_tokens=1073741825"} {
		t.Run(value, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, Filename), []byte("[agent]\n"+value+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(directory); err == nil {
				t.Fatal("invalid inference setting accepted")
			}
		})
	}
}

func TestInferenceAcceptsLargeExplicitOutput(t *testing.T) {
	directory := t.TempDir()
	c := Defaults()
	c.Agent.MaxOutputTokens = 196608
	if err := Save(directory, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(directory)
	if err != nil || loaded.Agent.MaxOutputTokens != 196608 {
		t.Fatalf("explicit output lost: %+v %v", loaded.Agent, err)
	}
}
