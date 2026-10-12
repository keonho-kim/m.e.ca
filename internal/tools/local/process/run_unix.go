//go:build linux || darwin

// Purpose: Run bounded shell commands and clean up their process groups on cancellation.
// Pattern: Process Adapter.
// Usage: Invoked for policy-approved execute_command operations in system Scope.
// Related: internal/tools/local/command.go, internal/tools/local/process/output.go, internal/tools/local/process/run_unsupported.go.

package process

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type cappedOutput struct {
	buffer    bytes.Buffer
	truncated bool
}

func (output *cappedOutput) Write(content []byte) (int, error) {
	count := len(content)
	remaining := 64*1024 - output.buffer.Len()
	if len(content) > remaining {
		content = content[:remaining]
		output.truncated = true
	}
	_, err := output.buffer.Write(content)
	return count, err
}

func Run(ctx context.Context, project, home, commandText string, timeoutSeconds int) (Output, error) {
	seconds := timeoutSeconds
	if seconds == 0 {
		seconds = 120
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "/bin/sh", "-c", commandText)
	command.Dir = project
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "LANG=C.UTF-8"}
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
	runErr := command.Run()
	if command.Process != nil {
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			runErr = errors.Join(runErr, err)
		}
	}
	if ctx.Err() != nil {
		return Output{}, ctx.Err()
	}
	exitCode := 0
	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			return Output{}, runErr
		}
		exitCode = exitErr.ExitCode()
	}
	return Output{Stdout: stdout.buffer.String(), Stderr: stderr.buffer.String(), ExitCode: exitCode, Truncated: stdout.truncated || stderr.truncated}, nil
}
