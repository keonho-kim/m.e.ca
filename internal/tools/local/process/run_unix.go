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

// start launches synchronously; its owner joins once, then materializes the capture.
func start(ctx context.Context, directory, commandText string, environment []string, stdout, stderr *cappedOutput) (startedCommand, error) {
	if !filepath.IsAbs(directory) {
		return startedCommand{}, fmt.Errorf("command working directory must be absolute")
	}
	command := exec.CommandContext(ctx, "/bin/sh", "-c", commandText)
	command.Dir = directory
	command.Env = environment
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
	command.Stdout, command.Stderr = stdout, stderr
	if err := command.Start(); err != nil {
		return startedCommand{}, err
	}
	var executionCause error
	wait := func() error {
		runErr := command.Wait()
		if ctx.Err() != nil {
			executionCause = context.Cause(ctx)
		}
		if command.Process != nil {
			if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				runErr = errors.Join(runErr, err)
			}
		}
		return runErr
	}
	output := func(runErr error) (Output, error) {
		outCapture, errCapture := stdout.snapshot(0), stderr.snapshot(0)
		output := Output{Revisions: [2]uint64{outCapture.Revision, errCapture.Revision}, Started: true, Stdout: outCapture.Text, Stderr: errCapture.Text, ExitCode: -1, Truncated: stdout.isTruncated() || stderr.isTruncated(), Status: "execution_failed"}
		if err := errors.Join(stdout.captureError(), stderr.captureError()); err != nil {
			return output, errors.Join(runErr, err)
		}
		if executionCause != nil {
			cause := executionCause
			output.Status = "interrupted"
			if errors.Is(executionCause, context.DeadlineExceeded) {
				output.Status = "timeout"
				cause = context.DeadlineExceeded
			}
			return output, cause
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
	return startedCommand{wait: wait, output: output}, nil
}
