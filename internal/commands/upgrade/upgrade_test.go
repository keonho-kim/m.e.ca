// Purpose: Verify failed upgrades preserve the installation and always remove private staging files.
// Pattern: Filesystem and HTTPS use-case integration tests.
// Usage: go test ./internal/commands/upgrade -run TestUpgradeFailure.
// Related: internal/commands/upgrade/upgrade.go, internal/commands/upgrade/fixtures_test.go, internal/commands/install/install.go.
package upgrade

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/install"
)

func installedFixture(t *testing.T) string {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(t.TempDir(), "previous")
	if err := os.WriteFile(old, []byte("previous installed binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := install.Install(context.Background(), install.Options{Home: home, Executable: old, Shell: "/bin/sh", Path: "/usr/bin:/bin"}); err != nil {
		t.Fatal(err)
	}
	return home
}

func assertNoStaging(t *testing.T, home string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(home, ".meca/.upgrade-*"))
	if err != nil || len(paths) != 0 {
		t.Fatalf("upgrade staging was not removed: %v, %v", paths, err)
	}
}

func TestUpgradeFailurePreservesInstallation(t *testing.T) {
	archive := tarball(t, archiveEntry{name: "meca", content: "not an executable format"})
	for _, scenario := range []string{"checksum", "missing asset", "network", "cancelled", "invalid executable"} {
		t.Run(scenario, func(t *testing.T) {
			home := installedFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := releaseFixture(t, archive, func(release *release, manifest *string) {
				switch scenario {
				case "checksum":
					*manifest = strings.Repeat("0", 64) + (*manifest)[64:]
				case "missing asset":
					release.Assets = release.Assets[:1]
				}
			}, func(writer http.ResponseWriter, request *http.Request) bool {
				if scenario == "cancelled" && request.URL.Path == "/releases/assets/2" {
					cancel()
					<-request.Context().Done()
					return true
				}
				if scenario == "network" {
					writer.WriteHeader(http.StatusServiceUnavailable)
					return true
				}
				return false
			})
			if _, err := Run(ctx, home, client); err == nil {
				t.Fatal("failed upgrade reported success")
			}
			body, err := os.ReadFile(install.BinaryPath(home))
			if err != nil || string(body) != "previous installed binary" {
				t.Fatalf("failed upgrade changed the installed executable: %v", err)
			}
			assertNoStaging(t, home)
		})
	}
}
