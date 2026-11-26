// Purpose: Search the installed development skill through its owned shell command.
// Pattern: Command use case with leased application state and joined process lifetime.
// Usage: cmd/meca dispatches meca develop before conversation or terminal initialization.
// Related: internal/commands/startup/parse.go, internal/config/statefs/lease_unix.go, internal/config/bootstrap/defaults/development/scripts/search.sh.
package develop

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/keonho-kim/meca/internal/commands/install"
	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

// Run streams search results while holding the installation's shared home lease.
func Run(ctx context.Context, home, query string, stdout, stderr io.Writer) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	lease, err := statefs.Lease(root, false)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	if err := install.CheckReady(home); err != nil {
		return err
	}
	const script = ".meca/skills/development/scripts/search.sh"
	info, err := root.Lstat(script)
	if err != nil {
		return faults.Errorf(diagcatalog.DevelopSearchUnavailable, filepath.Join(home, script), err)
	}
	if !info.Mode().IsRegular() {
		return faults.Errorf(diagcatalog.DevelopSearchMustBeRegular, filepath.Join(home, script))
	}
	command := exec.CommandContext(ctx, "/bin/sh", filepath.Join(home, script), query)
	command.Dir = home
	command.Env = append(os.Environ(), "HOME="+home)
	command.Stdout, command.Stderr = stdout, stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		} else {
			return err
		}
	}
	command.WaitDelay = 2 * time.Second
	runErr := command.Run()
	if ctx.Err() != nil {
		runErr = errors.Join(context.Cause(ctx), runErr)
	}
	if command.Process != nil {
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			runErr = errors.Join(runErr, err)
		}
	}
	if runErr != nil {
		return faults.Errorf(diagcatalog.DevelopSearchFailed, runErr)
	}
	return nil
}
