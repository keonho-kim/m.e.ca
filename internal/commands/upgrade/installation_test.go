// Purpose: Exercise both distribution switches and repeated upgrades while preserving configuration and sessions.
// Pattern: Executable integration test with a local HTTPS release.
// Usage: go test ./internal/commands/upgrade -run TestUpgradeInstallsReleaseAndRefreshesPrompts.
// Related: internal/commands/upgrade/upgrade.go, internal/commands/install/install.go, internal/config/prompts/refresh.go.
package upgrade

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/commands/install"
	"github.com/keonho-kim/meca/internal/distribution"
)

func TestUpgradeInstallsReleaseAndRefreshesPrompts(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the release installer")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	binaries := make(map[distribution.Kind]string)
	for _, kind := range []distribution.Kind{distribution.TUI, distribution.Full} {
		binary := filepath.Join(t.TempDir(), "meca")
		tags := "godbus_no_tcp"
		if kind == distribution.TUI {
			tags += ",tui"
		}
		build := exec.CommandContext(ctx, "go", "build", "-tags="+tags, "-o", binary, "../../../cmd/meca")
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", kind, err, output)
		}
		binaries[kind] = binary
	}
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
	// Return to TUI after full to cover both directions of binary replacement.
	for _, kind := range []distribution.Kind{distribution.TUI, distribution.Full, distribution.TUI} {
		body, err := os.ReadFile(binaries[kind])
		if err != nil {
			t.Fatal(err)
		}
		client := releaseFixture(t, kind, tarball(t, archiveEntry{name: "meca", content: string(body)}), nil, nil)
		for pass := 0; pass < 2; pass++ {
			for _, name := range []string{"system.md", "development.md", "compaction.md"} {
				if err := os.WriteFile(filepath.Join(home, ".meca/prompts/single", name), []byte("user edit"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			result, err := Run(ctx, home, client, kind)
			if err != nil || result.Version != "v0.0.20261026" || result.AlreadyCurrent != (pass == 1) {
				t.Fatalf("upgrade %d: %+v, %v", pass, result, err)
			}
			defaults := prompt.Defaults()
			for name, expected := range defaults.Files() {
				got, err := os.ReadFile(filepath.Join(home, ".meca/prompts/single", name))
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
			if output, err := command.CombinedOutput(); err != nil || !strings.Contains(string(output), "M.E.CA ("+string(kind)+")") {
				t.Fatalf("installed release is not %s: %v\n%s", kind, err, output)
			}
			assertNoStaging(t, home)
		}
	}
}
