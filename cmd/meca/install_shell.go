// Purpose: Replace a completed interactive installer with the user's login shell and activated PATH.
// Pattern: Unix process handoff adapter.
// Usage: maintenance calls activateInstallShell only after installation resources are closed.
// Related: cmd/meca/install.go, internal/commands/install/install.go, cmd/meca/install_shell_test.go.
package main

import (
	"context"

	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func activateInstallShell(ctx context.Context, shell, binary string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := exec.LookPath(shell)
	if err != nil {
		return faults.Errorf(diagcatalog.MecaInstallationCompletedButTheLoginShellIs, err)
	}
	activePath := filepath.Dir(binary)
	if previous := os.Getenv("PATH"); previous != "" {
		activePath += string(os.PathListSeparator) + previous
	}
	if err := os.Setenv("PATH", activePath); err != nil {
		return err
	}
	if err := syscall.Exec(path, []string{shell, "-l"}, os.Environ()); err != nil {
		return faults.Errorf(diagcatalog.MecaInstallationCompletedButLoginShellActivationFailed, err)
	}
	return nil
}
