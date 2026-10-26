// Purpose: Replace a completed interactive installer with the user's login shell and activated PATH.
// Pattern: Unix process handoff adapter.
// Usage: maintenance calls activateInstallShell only after installation resources are closed.
// Related: cmd/meca/install.go, internal/commands/install/install.go, cmd/meca/install_shell_test.go.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func activateInstallShell(ctx context.Context, shell, binary string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := exec.LookPath(shell)
	if err != nil {
		return fmt.Errorf("installation completed, but the login shell is unavailable: %w", err)
	}
	activePath := filepath.Dir(binary)
	if previous := os.Getenv("PATH"); previous != "" {
		activePath += string(os.PathListSeparator) + previous
	}
	if err := os.Setenv("PATH", activePath); err != nil {
		return err
	}
	if err := syscall.Exec(path, []string{shell, "-l"}, os.Environ()); err != nil {
		return fmt.Errorf("installation completed, but login shell activation failed: %w", err)
	}
	return nil
}
