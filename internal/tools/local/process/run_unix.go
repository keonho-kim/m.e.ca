//go:build linux || darwin

// Purpose: Run shell commands under the caller context and clean up process groups on cancellation.
// Pattern: Process Adapter.
// Usage: Approved calls supply the project directory, command and prepared environment snapshot.
// Related: internal/tools/local/command.go, internal/tools/local/process/output.go, internal/tools/local/process/run_unsupported.go.

package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func Run(ctx context.Context, directory, commandText string, environment []string) (Output, error) {
	if !filepath.IsAbs(directory) {
		return Output{Status: "execution_failed", ExitCode: -1}, fmt.Errorf("command working directory must be absolute")
	}
	command := exec.CommandContext(ctx, "/bin/sh", "-c", commandText)
	command.Dir = directory
	command.Env = append([]string{}, environment...)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	command.WaitDelay = 2 * time.Second
	var stdout, stderr cappedOutput
	command.Stdout, command.Stderr = &stdout, &stderr
	runErr := command.Start()
	started := runErr == nil
	if started {
		runErr = command.Wait()
	}
	if command.Process != nil {
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			runErr = errors.Join(runErr, err)
		}
	}
	output := Output{Started: started, Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: -1, Truncated: stdout.truncated || stderr.truncated, Status: "execution_failed"}
	if ctx.Err() != nil {
		output.Status = "interrupted"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			output.Status = "timeout"
		}
		return output, ctx.Err()
	}
	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			return output, runErr
		}
		output.ExitCode = exitErr.ExitCode()
	} else {
		output.ExitCode = 0
	}
	output.Status = "exited"
	return output, nil
}
