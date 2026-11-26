// Purpose: Verify complete uninstall, user-file protection and recovery after permission failures.
// Pattern: Filesystem lifecycle integration tests.
// Usage: go test ./internal/commands/install -run TestUninstall.
// Related: internal/commands/install/uninstall.go, internal/commands/install/removal.go.
package install

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func TestUninstallAfterReinstallPreservesShellSettings(t *testing.T) {
	o := options(t, "/bin/sh")
	profile := filepath.Join(o.Home, ".profile")
	original, added := "export EDITOR=vim", "\nexport PAGER=less\n"
	if err := os.WriteFile(profile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(profile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteString(added)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		t.Fatal(err)
	}
	result, err := Install(t.Context(), o)
	if err != nil || !result.RestartShell || result.Binary != BinaryPath(o.Home) {
		t.Fatalf("reinstall: %+v %v", result, err)
	}
	command := exec.CommandContext(t.Context(), "/bin/sh", "-ec", `. "$HOME/.profile"; printf '%s\n' "$PATH"`)
	command.Env = []string{"HOME=" + o.Home, "PATH=" + o.Path}
	output, err := command.CombinedOutput()
	if want := filepath.Dir(result.Binary) + ":" + o.Path + "\n"; err != nil || string(output) != want {
		t.Fatalf("shell PATH after reinstall: %q %v", output, err)
	}
	if err := Uninstall(t.Context(), o.Home); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(o.Home, ".meca"), BinaryPath(o.Home)} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("installation was not removed at %s: %v", path, err)
		}
	}
	if body, err := os.ReadFile(profile); err != nil || string(body) != original+added {
		t.Fatalf("user shell settings changed: %q %v", body, err)
	}
	if err := Uninstall(t.Context(), o.Home); err != nil {
		t.Fatalf("repeat uninstall: %v", err)
	}
}

func TestUninstallRefusesActiveState(t *testing.T) {
	o := options(t, "/bin/bash")
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	root, err := statefs.Open(filepath.Join(o.Home, ".meca"))
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	lease, err := statefs.Lease(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := Uninstall(t.Context(), o.Home); !errors.Is(err, syscall.EWOULDBLOCK) {
		t.Fatalf("uninstall accepted active state: %v", err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(t.Context(), o.Home); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".bashrc", ".profile"} {
		if _, err := os.Stat(filepath.Join(o.Home, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("created shell file left behind: %s", name)
		}
	}
}

func TestUninstallPreservesUnrelatedFiles(t *testing.T) {
	t.Run("replaced executable", func(t *testing.T) {
		o := options(t, "/bin/zsh")
		if _, err := Install(t.Context(), o); err != nil {
			t.Fatal(err)
		}
		path := BinaryPath(o.Home)
		if err := os.WriteFile(path, []byte("another program"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := Uninstall(t.Context(), o.Home); err == nil {
			t.Fatal("uninstall accepted an unrelated binary")
		}
		if body, err := os.ReadFile(path); err != nil || string(body) != "another program" {
			t.Fatalf("unrelated binary changed: %q %v", body, err)
		}
	})
	t.Run("redirected state", func(t *testing.T) {
		target, home := t.TempDir(), t.TempDir()
		path := filepath.Join(target, "user-file")
		if err := os.WriteFile(path, []byte("user data"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(home, ".meca")); err != nil {
			t.Fatal(err)
		}
		if err := Uninstall(t.Context(), home); err == nil {
			t.Fatal("uninstall accepted redirected state")
		}
		if body, err := os.ReadFile(path); err != nil || string(body) != "user data" {
			t.Fatalf("unrelated file changed: %q %v", body, err)
		}
	})
}

func TestUninstallPermissionRecovery(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires a user without root permission overrides")
	}
	o := options(t, "/bin/sh")
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(o.Home, ".meca", "blocked")
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(blocked, "retained")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(blocked, 0700); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Error(err)
		}
	})
	if err := os.Chmod(blocked, 0500); err != nil {
		t.Fatal(err)
	}
	err := Uninstall(t.Context(), o.Home)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected removal permission failure: %v", err)
	}
	message := fault.PublicMessage(err, func(value string) string { return value })
	if !strings.Contains(message, ".meca") || !strings.Contains(message, "permission") {
		t.Errorf("permission failure lacks its path or repair guidance: %s", message)
	}
	if body, err := os.ReadFile(file); err != nil || string(body) != "fixture" {
		t.Fatalf("blocked state changed: %q %v", body, err)
	}
	if _, err := Install(t.Context(), o); err == nil {
		t.Fatal("reinstall accepted incomplete removal")
	}
	if err := os.Chmod(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(t.Context(), o.Home); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatalf("reinstall after permission recovery: %v", err)
	}
}
