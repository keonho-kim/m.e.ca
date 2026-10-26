// Purpose: Keep supplied work prompts free of agent-orchestration explanations.
// Pattern: Prompt contract regression tests.
// Usage: go test ./internal/agent/prompt -run TestTaskPrompts.
// Related: internal/agent/prompt/defaults.go, internal/agent/prompt/prompt.go.
package prompt

import (
	"strings"
	"testing"
)

func TestTaskPromptsDoNotDescribeHarness(t *testing.T) {
	templates := Defaults()
	for name, text := range templates.Files() {
		for _, term := range []string{"harness", "checkpoint", "runtime"} {
			if strings.Contains(strings.ToLower(*text), term) {
				t.Errorf("%s exposes orchestration term %q", name, term)
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
