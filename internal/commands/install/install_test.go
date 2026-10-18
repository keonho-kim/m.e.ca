// Purpose: Verify repeatable installation, scoped removal and protection against active or unrelated state.
// Pattern: Filesystem lifecycle integration tests.
// Usage: go test ./internal/commands/install.
// Related: internal/commands/install/install.go, internal/commands/install/uninstall.go.
package install

import (
	"context"
	"errors"
	"github.com/keonho-kim/v-code/internal/config/statefs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func options(t *testing.T, shell string) Options {
	t.Helper()
	home := t.TempDir()
	executable := filepath.Join(t.TempDir(), "v")
	if err := os.WriteFile(executable, []byte("test executable"), 0755); err != nil {
		t.Fatal(err)
	}
	return Options{Home: home, Executable: executable, Shell: shell, Path: "/usr/bin:/bin"}
}

func TestCredentialFailurePreservesInstallationForRetry(t *testing.T) {
	o := options(t, "/bin/zsh")
	if _, err := Install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(o.Home, ".v/credentials.used"), []byte("used"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), o.Home, func() error { return errors.New("keyring locked") }); err == nil {
		t.Fatal("ignored credential cleanup failure")
	}
	for _, name := range []string{binaryName, ".v/installation.json", ".zshrc"} {
		if _, err := os.Stat(filepath.Join(o.Home, name)); err != nil {
			t.Fatalf("cleanup failure destroyed %s: %v", name, err)
		}
	}
}

func TestUserChangesAfterInstallArePreserved(t *testing.T) {
	o := options(t, "/bin/zsh")
	if _, err := Install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(o.Home, ".zshrc")
	file, err := os.OpenFile(profile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("export EDITOR=vim\n"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), o.Home, nil); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(profile)
	if err != nil || string(body) != "export EDITOR=vim\n" {
		t.Fatalf("user edit lost: %q %v", body, err)
	}
}

func TestInstallAndCleanUninstall(t *testing.T) {
	ctx := context.Background()
	o := options(t, "/bin/zsh")
	original := []byte("export EDITOR=vim")
	profile := filepath.Join(o.Home, ".zshrc")
	if err := os.WriteFile(profile, original, 0644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		result, err := Install(ctx, o)
		if err != nil || !result.RestartShell {
			t.Fatalf("%+v %v", result, err)
		}
	}
	body, err := os.ReadFile(profile)
	if err != nil || strings.Count(string(body), pathBlock) != 1 {
		t.Fatalf("duplicated registration: %q %v", body, err)
	}
	for _, name := range []string{"system.md", "development.md", "compaction.md"} {
		if _, err := os.Stat(filepath.Join(o.Home, ".v/prompts", name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(o.Home, ".v/credentials.used"), []byte("used"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	if err := Uninstall(ctx, o.Home, func() error { calls++; return nil }); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("credential cleanup omitted")
	}
	for _, path := range []string{".v", binaryName} {
		if _, err := os.Lstat(filepath.Join(o.Home, path)); !os.IsNotExist(err) {
			t.Fatalf("leftover %s: %v", path, err)
		}
	}
	body, err = os.ReadFile(profile)
	if err != nil || string(body) != string(original) {
		t.Fatalf("shell changes lost: %q %v", body, err)
	}
	if err := Uninstall(ctx, o.Home, nil); err != nil {
		t.Fatal(err)
	}
}

func TestUninstallRefusesActiveState(t *testing.T) {
	o := options(t, "/bin/bash")
	if _, err := Install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	root, err := statefs.Open(filepath.Join(o.Home, ".v"))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	lease, err := statefs.Lease(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), o.Home, nil); err == nil {
		t.Fatal("deleted active state")
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), o.Home, nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".bashrc", ".profile"} {
		if _, err := os.Stat(filepath.Join(o.Home, name)); !os.IsNotExist(err) {
			t.Fatalf("created shell file left behind: %s", name)
		}
	}
}

func TestUninstallProtectsReplacedBinaryAndSymlinkState(t *testing.T) {
	o := options(t, "/bin/zsh")
	if _, err := Install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(o.Home, binaryName), []byte("another program"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), o.Home, nil); err == nil {
		t.Fatal("removed unrelated binary")
	}
	target := t.TempDir()
	home := t.TempDir()
	if err := os.Symlink(target, filepath.Join(home, ".v")); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), home, nil); err == nil {
		t.Fatal("followed state symlink")
	}
}
