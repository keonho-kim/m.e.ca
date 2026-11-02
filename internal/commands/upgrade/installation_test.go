// Purpose: Exercise release upgrade through a real installer while preserving configuration and sessions.
// Pattern: Executable integration test with a local HTTPS release.
// Usage: go test ./internal/commands/upgrade -run TestUpgradeInstallsReleaseAndRefreshesPrompts.
// Related: internal/commands/upgrade/upgrade.go, internal/commands/install/install.go, internal/config/prompts/refresh.go.
package upgrade

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/commands/install"
)

func TestUpgradeInstallsReleaseAndRefreshesPrompts(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the release installer")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "meca")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../../cmd/meca")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	body, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	client := releaseFixture(t, tarball(t, archiveEntry{name: "meca", content: string(body)}), nil, nil)
	home := installedFixture(t)
	t.Setenv("SHELL", "/bin/sh")
	configuration := filepath.Join(home, ".meca/config.toml")
	before, err := os.ReadFile(configuration)
	if err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(home, ".meca/sessions/retained.db")
	if err := os.WriteFile(session, []byte("retained session"), 0600); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		for _, name := range []string{"system.md", "development.md", "compaction.md"} {
			if err := os.WriteFile(filepath.Join(home, ".meca/prompts", name), []byte("user edit"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		result, err := Run(ctx, home, client)
		if err != nil || result.Version != "v0.0.20261026" || result.AlreadyCurrent != (pass == 1) {
			t.Fatalf("upgrade %d: %+v, %v", pass, result, err)
		}
		defaults := prompt.Defaults()
		for name, expected := range defaults.Files() {
			got, err := os.ReadFile(filepath.Join(home, ".meca/prompts", name))
			if err != nil || string(got) != *expected {
				t.Fatalf("upgrade did not refresh %s: %v", name, err)
			}
		}
		after, err := os.ReadFile(configuration)
		if err != nil || string(after) != string(before) {
			t.Fatalf("upgrade changed configuration: %v", err)
		}
		got, err := os.ReadFile(session)
		if err != nil || string(got) != "retained session" {
			t.Fatalf("upgrade changed session data: %v", err)
		}
		command := exec.CommandContext(ctx, install.BinaryPath(home), "--help")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("installed release cannot execute: %v\n%s", err, output)
		}
		assertNoStaging(t, home)
	}
}
