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

func TestInferenceDefaultsAndOverrides(t *testing.T) {
	directory := t.TempDir()
	if err := Save(directory, Defaults()); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(directory, Filename))
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"agent", "plan", "conclave", "compaction", "diagnostic"} {
		if !strings.Contains(string(body), "[inference."+role+"]") {
			t.Fatalf("missing persisted role %s", role)
		}
	}
	loaded, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []inference.Role{inference.Agent, inference.Plan, inference.Conclave, inference.Compaction, inference.Diagnostic} {
		parameters, err := inference.Resolve(&loaded.Inference, role)
		want := 12800
		if role == inference.Agent {
			want = 25600
		}
		if err != nil || parameters.MaxOutputTokens != want {
			t.Fatalf("persisted %s default=%+v, want %d: %v", role, parameters, want, err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, Filename), []byte("[inference.conclave]\ntemperature=0.0\nmax_output_tokens=1536\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	if c.Inference.Conclave.Temperature != 0 || c.Inference.Conclave.MaxOutputTokens != 1536 || c.Inference.Agent.Temperature != 0.7 || c.Inference.Compaction.Temperature != 0.2 {
		t.Fatalf("role override lost: %+v", c.Inference)
	}
}
func TestInferenceRejectsInvalidSettings(t *testing.T) {
	for _, value := range []string{"temperature=-1", "temperature=nan", "temperature=inf", "max_output_tokens=0", "max_output_tokens=65537"} {
		t.Run(value, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, Filename), []byte("[inference.agent]\n"+value+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(directory); err == nil {
				t.Fatal("invalid inference setting accepted")
			}
		})
	}
}
