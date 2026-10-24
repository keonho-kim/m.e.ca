// Purpose: Verify the last model connection persists without credentials or discovery snapshots.
// Pattern: Configuration round-trip test.
// Usage: go test ./internal/config -run TestSavedModel.
// Related: internal/config/model.go, internal/config/config.go.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestSavedModelRoundTrip(t *testing.T) {
	conn, err := connection.New(connection.VLLM, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	conn.CredentialRef = "credential_saved"
	cfg := Defaults()
	on, effort := true, "high"
	cfg.Model = RememberModel(connection.Selection{Connection: conn, Model: "models/bin/modelX", Thinking: &on, ReasoningEffort: &effort})
	dir := t.TempDir()
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := restored.Model.Selection()
	if err != nil || selected.Connection != conn || selected.Model != "models/bin/modelX" {
		t.Fatalf("%+v %v", selected, err)
	}
	if selected.Thinking == nil || !*selected.Thinking || selected.Effort() != effort {
		t.Fatalf("model preferences lost: %+v", selected)
	}
	content, err := os.ReadFile(filepath.Join(dir, Filename))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "credential_ref = 'credential_saved'") || strings.Contains(string(content), "api_key") {
		t.Fatalf("unexpected connection config: %s", content)
	}
	cfg.Model.Endpoint = "http://user:secret@localhost:8000"
	if err := Save(dir, cfg); err == nil {
		t.Fatal("persisted endpoint credentials")
	}
}
