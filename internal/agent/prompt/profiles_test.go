// Purpose: Verify team prompts inherit shared instructions and compaction defaults.
// Pattern: Prompt composition tests.
// Usage: go test ./internal/agent/prompt -run TestTeamSystem.
// Related: internal/agent/prompt/defaults.go, internal/agent/prompt/prompt.go, internal/config/prompts/profiles_test.go.
package prompt

import (
	"strings"
	"testing"
)

func TestTeamSystemExtendsSingleInstructionsExactlyOnce(t *testing.T) {
	single, team := DefaultsFor(Single), DefaultsFor(Team)
	if !strings.HasPrefix(team.System, single.System+"\n") {
		t.Fatal("team system replaced the common instructions instead of extending them")
	}
	if strings.Count(team.System, single.System) != 1 {
		t.Fatal("team system repeated the common instructions")
	}
	if strings.TrimSpace(strings.TrimPrefix(team.System, single.System)) == "" {
		t.Fatal("team system has no collaboration guidance")
	}
	if team.Compaction != single.Compaction {
		t.Fatal("compaction defaults diverged between modes")
	}
}
