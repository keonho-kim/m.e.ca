// Purpose: Verify mounted-state removal and recovery against real Linux filesystems.
// Pattern: Filesystem lifecycle integration tests with test-owned tmpfs mounts.
// Usage: go test ./internal/commands/install; mount cases require CAP_SYS_ADMIN.
// Related: internal/commands/install/removal.go, internal/config/credentials/persistence.go.
package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"golang.org/x/sys/unix"
)

func mountTestState(t *testing.T, directory string) func() {
	t.Helper()
	if testing.Short() {
		t.Skip("requires the Linux mount integration suite")
	}
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mount("tmpfs", directory, "tmpfs", 0, "mode=0700"); err != nil {
		if errors.Is(err, unix.EPERM) || errors.Is(err, unix.EACCES) {
			t.Skip("requires permission to mount a test-owned tmpfs", err)
		}
		t.Fatal(err)
	}
	mounted := true
	unmount := func() {
		t.Helper()
		if mounted {
			if err := unix.Unmount(directory, 0); err != nil {
				t.Fatal(err)
			}
			mounted = false
		}
	}
	t.Cleanup(unmount)
	return unmount
}

func TestUninstallMountedStateAllowsReinstall(t *testing.T) {
	o := options(t, "/bin/sh")
	state := filepath.Join(o.Home, ".meca")
	mountTestState(t, state)
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	vault := &credentials.Vault{State: state}
	if err := vault.Set(t.Context(), "provider", "scope", "fixture-api-key"); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(t.Context(), o.Home); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(state); err != nil || len(entries) != 0 {
		t.Fatalf("mounted state was not emptied: %v %v", entries, err)
	}
	if _, err := os.Stat(BinaryPath(o.Home)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("installed binary was not removed: %v", err)
	}
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatalf("reinstall on the retained mount: %v", err)
	}
}

func TestUninstallNestedMountRecovery(t *testing.T) {
	o := options(t, "/bin/sh")
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	unmount := mountTestState(t, filepath.Join(o.Home, ".meca", "nested"))
	if err := Uninstall(t.Context(), o.Home); !errors.Is(err, unix.EBUSY) {
		t.Fatalf("expected nested mount removal failure: %v", err)
	}
	if _, err := Install(t.Context(), o); err == nil {
		t.Fatal("reinstall accepted incomplete removal")
	}
	unmount()
	if err := Uninstall(t.Context(), o.Home); err != nil {
		t.Fatalf("removal after unmount: %v", err)
	}
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatalf("reinstall after unmount: %v", err)
	}
}

func TestUninstallReadOnlyStateRecovery(t *testing.T) {
	o := options(t, "/bin/sh")
	state := filepath.Join(o.Home, ".meca")
	mountTestState(t, state)
	if _, err := Install(t.Context(), o); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mount("", state, "", unix.MS_REMOUNT|unix.MS_RDONLY, ""); err != nil {
		t.Fatal(err)
	}
	err := Uninstall(t.Context(), o.Home)
	if !errors.Is(err, unix.EROFS) {
		t.Fatalf("expected read-only filesystem failure: %v", err)
	}
	message := fault.PublicMessage(err, func(value string) string { return value })
	if !strings.Contains(message, state) || !strings.Contains(message, "read-only") {
		t.Errorf("read-only failure lacks its path or repair guidance: %s", message)
	}
	if err := unix.Mount("", state, "", unix.MS_REMOUNT, ""); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(t.Context(), o.Home); err != nil {
		t.Fatalf("removal after restoring write access: %v", err)
	}
}
