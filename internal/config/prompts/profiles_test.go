// Purpose: Verify complete shared-base profile defaults are published and remain independently editable.
// Pattern: Filesystem contract tests.
// Usage: go test ./internal/config/prompts -run TestProfiles.
// Related: internal/config/prompts/prompts.go, internal/agent/prompt/defaults.go.
package prompts_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/prompts"
)

func TestProfilesBootstrapAndLoadSeparately(t *testing.T) {
	home := t.TempDir()
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, ".meca")
	for _, profile := range []prompt.Profile{prompt.Single, prompt.Team} {
		defaults := prompt.DefaultsFor(profile)
		for name, expected := range defaults.Files() {
			body, err := os.ReadFile(filepath.Join(state, "prompts", string(profile), name))
			if err != nil || string(body) != *expected {
				t.Fatalf("%s/%s did not publish the complete default: %v", profile, name, err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(state, "prompts", "team", "system.md"), []byte("Team custom instructions"), 0600); err != nil {
		t.Fatal(err)
	}
	team, err := prompts.LoadProfile(state, prompt.Team)
	if err != nil || team.System != "Team custom instructions" {
		t.Fatalf("team: %v %v", team, err)
	}
	single, err := prompts.Load(state)
	if err != nil || single.System != prompt.Defaults().System {
		t.Fatalf("single: %v %v", single, err)
	}
}
