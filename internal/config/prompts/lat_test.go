// Purpose: Verify bootstrap publishes LAT and inactive requests do not read its prompt.
// Pattern: Private filesystem contract tests.
// Usage: go test ./internal/config/prompts -run TestLAT.
// Related: internal/config/prompts/prompts.go, internal/config/prompts/refresh.go.
package prompts_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/prompts"
	"github.com/keonho-kim/meca/internal/config/statefs"
)

func TestLATPromptIsLoadedOnlyWhenEnabledAndRefreshed(t *testing.T) {
	home := t.TempDir()
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, ".meca")
	path := filepath.Join(state, "prompts", "lat.md")
	text, err := prompts.LoadLAT(state)
	if err != nil || text.LAT != prompt.Defaults().LAT || text.LATCompaction != prompt.Defaults().LATCompaction {
		t.Fatalf("missing LAT bootstrap default: %v", err)
	}
	if err := os.WriteFile(path, []byte("# Custom review\nInspect requirements carefully."), 0600); err != nil {
		t.Fatal(err)
	}
	text, err = prompts.LoadLAT(state)
	if err != nil || text.LAT == prompt.Defaults().LAT {
		t.Fatalf("custom LAT prompt ignored: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(state, "prompts", "lat_compaction.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := prompts.Load(state); err != nil {
		t.Fatal("inactive LAT changed normal prompt loading", err)
	}
	if _, err := prompts.LoadLAT(state); err == nil {
		t.Fatal("enabled LAT silently accepted a missing prompt")
	}
	root, err := statefs.Open(state)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := prompts.Refresh(root); err != nil {
		t.Fatal(err)
	}
	text, err = prompts.LoadLAT(state)
	if err != nil || text.LAT != prompt.Defaults().LAT || text.LATCompaction != prompt.Defaults().LATCompaction {
		t.Fatalf("installation did not refresh LAT: %v", err)
	}
}
