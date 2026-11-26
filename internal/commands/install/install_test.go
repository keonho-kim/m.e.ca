// Purpose: Verify extension preparation and configuration-preserving reinstall.
// Pattern: Filesystem lifecycle integration tests.
// Usage: go test ./internal/commands/install.
// Related: internal/commands/install/install.go, internal/commands/install/uninstall.go.
package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/keonho-kim/meca/internal/config/statefs"
)

func options(t *testing.T, shell string) Options {
	t.Helper()
	home := t.TempDir()
	executable := filepath.Join(t.TempDir(), "meca")
	if err := os.WriteFile(executable, []byte("test executable"), 0755); err != nil {
		t.Fatal(err)
	}
	return Options{Home: home, Executable: executable, Shell: shell, Path: "/usr/bin:/bin"}
}

func TestInstallPreparesExtensionAfterBase(t *testing.T) {
	o := options(t, "/bin/sh")
	prepared := false
	o.PrepareExtension = func(ctx context.Context, state string) error {
		if state != filepath.Join(o.Home, statefs.DirectoryName) {
			t.Fatalf("unexpected extension state: %s", state)
		}
		for _, name := range []string{BinaryPath(o.Home), filepath.Join(state, "config.toml"), filepath.Join(o.Home, ".profile")} {
			if _, err := os.Stat(name); err != nil {
				t.Fatal(err)
			}
		}
		if err := Uninstall(ctx, o.Home); !errors.Is(err, syscall.EWOULDBLOCK) {
			t.Fatalf("uninstall during extension preparation: %v", err)
		}
		prepared = true
		return nil
	}
	if _, err := Install(t.Context(), o); err != nil || !prepared {
		t.Fatalf("extension preparation completed=%t: %v", prepared, err)
	}
}

func TestReinstallRefreshesEditedPromptsAndPreservesConfig(t *testing.T) {
	o := options(t, "/bin/zsh")
	ctx := context.Background()
	if _, err := Install(ctx, o); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"system.md", "compaction.md"} {
		if err := os.WriteFile(filepath.Join(o.Home, ".meca/prompts/single", name), []byte("Custom prompt"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	configuration := filepath.Join(o.Home, ".meca/config.toml")
	before, err := os.ReadFile(configuration)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install(ctx, o); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"system.md", "compaction.md"} {
		body, err := os.ReadFile(filepath.Join(o.Home, ".meca/prompts/single", name))
		if err != nil || string(body) == "Custom prompt" {
			t.Fatalf("reinstall did not refresh %s: %v", name, err)
		}
	}
	after, err := os.ReadFile(configuration)
	if err != nil || string(before) != string(after) {
		t.Fatalf("reinstall changed configuration: %v", err)
	}
}
