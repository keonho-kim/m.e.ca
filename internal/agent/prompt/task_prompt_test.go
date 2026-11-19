// Purpose: Verify task-only guidance and common system instruction inheritance across modes.
// Pattern: Prompt contract regression tests.
// Usage: go test ./internal/agent/prompt -run TestTaskPrompts.
// Related: internal/agent/prompt/defaults.go, internal/agent/prompt/prompt.go, internal/config/prompts/profiles_test.go.
package prompt

import (
	"strings"
	"testing"
)

func TestTaskPromptsDoNotDescribeHarness(t *testing.T) {
	for _, profile := range []Profile{Single, Team} {
		templates := DefaultsFor(profile)
		for name, text := range templates.Files() {
			for _, term := range []string{"harness", "checkpoint", "runtime"} {
				if strings.Contains(strings.ToLower(*text), term) {
					t.Errorf("%s/%s exposes orchestration term %q", profile, name, term)
				}
			}
		}
	}
	system, err := Compose("Work on the request.", "Keep the API.")
	if err != nil || !strings.HasPrefix(system, "Work on the request.") || !strings.Contains(system, "Keep the API.") {
		t.Fatal("lost supplied work instructions")
	}
	if strings.Contains(system, "completion review") || strings.Contains(system, "native function") {
		t.Fatal("hidden orchestration instructions appended")
	}
}

func TestTeamSystemIsConciseEnglishGuidance(t *testing.T) {
	text := DefaultsFor(Team).System
	if lines := strings.Count(text, "\n"); lines > 100 {
		t.Fatalf("team system exceeds the 100-line budget: %d", lines)
	}
	for _, r := range text {
		if r > 127 {
			t.Fatalf("team system prompt must be authored in English: %q", r)
		}
	}
}

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
	if team.Development == single.Development {
		t.Fatal("team development must have its own default")
	}
	if team.Compaction != single.Compaction {
		t.Fatal("compaction defaults diverged between modes")
	}
}

func TestSystemGuidesUserLanguageWithoutImplementationDetails(t *testing.T) {
	system := Defaults().System
	for _, rule := range []string{
		"You are M.E.CA (Minimal Essentials Coding Agent)",
		"When asked your name, introduce yourself as M.E.CA.",
		"language of the user's latest request", "Do not write code while thinking", "completed changes, actual checks",
	} {
		if !strings.Contains(system, rule) {
			t.Fatalf("missing work guideline %q", rule)
		}
	}
	for _, term := range []string{"SQLite", "model call budget", "checkpoint"} {
		if strings.Contains(system, term) {
			t.Fatalf("system describes internal implementation %q", term)
		}
	}
	for _, r := range system {
		if r > 127 {
			t.Fatalf("default system prompt must be authored in English: %q", r)
		}
	}
}
