// Purpose: Verify publication, edits and activation-only loading of every LAT prompt.
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

func TestLATPromptEditsSurviveBootstrapAndRefreshDefaults(t *testing.T) {
	home := t.TempDir()
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, ".meca")
	defaults := prompt.Defaults()
	for _, name := range latPromptNames {
		if err := os.WriteFile(filepath.Join(state, "prompts", name), []byte("# Custom "+name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	text, err := prompts.LoadLAT(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range latPromptNames {
		if got := *text.Files()[name]; got != "# Custom "+name {
			t.Fatalf("bootstrap lost custom %s: %q", name, got)
		}
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
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range latPromptNames {
		if got := *text.Files()[name]; got != *defaults.Files()[name] {
			t.Fatalf("installation did not refresh %s", name)
		}
	}
}

var latPromptNames = []string{"lat.md", "lat_plan.md", "lat_critique.md", "lat_refine.md", "lat_compaction.md"}

func TestLATPromptsLoadOnlyWhenEnabled(t *testing.T) {
	home := t.TempDir()
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, ".meca")
	defaults := prompt.Defaults()
	for _, name := range latPromptNames {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(state, "prompts", name)
			defer func() {
				if err := os.WriteFile(path, []byte(*defaults.Files()[name]), 0600); err != nil {
					t.Fatal(err)
				}
			}()
			if err := os.WriteFile(path, nil, 0600); err != nil {
				t.Fatal(err)
			}
			assertLATLoadIsolation(t, state)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			assertLATLoadIsolation(t, state)
		})
	}
}

func assertLATLoadIsolation(t *testing.T, state string) {
	t.Helper()
	text, err := prompts.Load(state)
	if err != nil {
		t.Fatal("inactive LAT changed normal prompt loading", err)
	}
	for _, name := range latPromptNames {
		if *text.Files()[name] != "" {
			t.Fatalf("inactive LAT loaded %s", name)
		}
	}
	if _, err := prompts.LoadLAT(state); err == nil {
		t.Fatal("enabled LAT silently accepted an invalid or missing prompt")
	}
}
