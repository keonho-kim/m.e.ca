// Purpose: Verify explicit interface language settings persist and unsupported values are rejected.
// Pattern: Configuration boundary tests.
// Usage: go test ./internal/config -run TestLanguage.
// Related: internal/config/config.go, internal/config/settings.go.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLanguageConfiguration(t *testing.T) {
	for _, value := range []string{"auto", "ko", "en"} {
		t.Run(value, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, Filename), []byte("language = \""+value+"\"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			configuration, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			if err := Save(root, configuration); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join(root, Filename))
			if err != nil || !strings.Contains(string(body), "language = '"+value+"'") {
				t.Fatalf("language not persisted: %s, %v", body, err)
			}
		})
	}
	for _, value := range []string{"", "kr", "ja", "EN"} {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, Filename), []byte("language = \""+value+"\"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(root); err == nil {
			t.Fatalf("unsupported language accepted: %q", value)
		}
	}
}
