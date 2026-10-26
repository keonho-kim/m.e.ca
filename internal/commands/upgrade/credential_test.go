// Purpose: Verify token precedence and optional GitHub CLI authentication without real credentials.
// Pattern: Isolated environment and executable fixtures.
// Usage: go test ./internal/commands/upgrade -run TestCredential.
// Related: internal/commands/upgrade/credential.go.
package upgrade

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialSources(t *testing.T) {
	for _, tc := range []struct{ name, primary, secondary, cli, want string }{
		{"GH_TOKEN first", "primary", "secondary", "exit 9", "primary"},
		{"GITHUB_TOKEN", "", "secondary", "exit 9", "secondary"},
		{"CLI login", "", "", "printf fixture-cli-token", "fixture-cli-token"},
		{"anonymous", "", "", "exit 1", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, "gh"), []byte("#!/bin/sh\n"+tc.cli+"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", directory)
			t.Setenv("GH_TOKEN", tc.primary)
			t.Setenv("GITHUB_TOKEN", tc.secondary)
			if got := Credential(context.Background()); got != tc.want {
				t.Fatalf("wrong credential source for %s", tc.name)
			}
		})
	}
}
