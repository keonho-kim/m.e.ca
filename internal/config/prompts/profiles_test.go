// Purpose: Verify distinct profile defaults are published and remain independently editable.
// Pattern: Filesystem contract tests.
// Usage: go test ./internal/config/prompts -run TestProfiles.
// Related: internal/config/prompts/prompts.go, internal/agent/prompt/defaults.go.
package prompts_test

import (
	"os"
	"path/filepath"
	"strings"
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
	singleDefaults, teamDefaults := prompt.Defaults(), prompt.DefaultsFor(prompt.Team)
	if teamDefaults.Development == singleDefaults.Development || !strings.HasPrefix(teamDefaults.Development, "# Team development workflow\n") {
		t.Fatal("team development did not select its independent workflow default")
	}
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
	if err := os.WriteFile(filepath.Join(state, "prompts", "team", "development.md"), []byte("Team custom development"), 0600); err != nil {
		t.Fatal(err)
	}
	team, err := prompts.LoadProfile(state, prompt.Team)
	if err != nil || team.System != "Team custom instructions" || team.Development != "Team custom development" {
		t.Fatalf("team: %v %v", team, err)
	}
	single, err := prompts.Load(state)
	if err != nil || single.System != singleDefaults.System || single.Development != singleDefaults.Development {
		t.Fatalf("single: %v %v", single, err)
	}
}

func TestRefreshUsesIndependentTeamDevelopmentDefault(t *testing.T) {
	home := t.TempDir()
	if _, err := bootstrap.Initialize(home); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, ".meca")
	path := filepath.Join(state, "prompts", "team", "development.md")
	if err := os.WriteFile(path, []byte("Team custom development"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(state)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := prompts.Refresh(root); err != nil {
		t.Fatal(err)
	}
	team, err := prompts.LoadProfile(state, prompt.Team)
	if err != nil || team.Development != prompt.DefaultsFor(prompt.Team).Development || team.Development == prompt.Defaults().Development {
		t.Fatalf("refresh replaced team development with the wrong default: %v", err)
	}
	single, err := prompts.Load(state)
	if err != nil || single.Development != prompt.Defaults().Development {
		t.Fatalf("refresh changed single development: %v", err)
	}
}
