// Purpose: Preserve server defaults and optional explicit global Thinking preferences in TOML.
// Pattern: Configuration persistence tests.
// Usage: go test ./internal/config -run TestThinkingPreference.
// Related: internal/config/settings.go, internal/config/config.go.
package config

import "testing"

func TestThinkingPreference(t *testing.T) {
	directory := t.TempDir()
	on, off := true, false
	for _, choice := range []*bool{nil, &on, &off, nil} {
		current := Defaults()
		current.Thinking = choice
		if err := Save(directory, current); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(directory)
		if err != nil {
			t.Fatal(err)
		}
		if (loaded.Thinking == nil) != (choice == nil) || choice != nil && *loaded.Thinking != *choice {
			t.Fatalf("choice=%v loaded=%v", choice, loaded.Thinking)
		}
	}
}
