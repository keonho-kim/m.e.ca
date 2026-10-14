//go:build linux || darwin

// Purpose: Run shell commands under the caller context and clean up process groups on cancellation.
// Pattern: Process Adapter.
// Usage: Invoked after policy and approval checks; project Scope requires an OS-isolated launch.
// Related: internal/tools/local/command.go, internal/tools/local/process/output.go, internal/tools/local/process/run_unsupported.go.

package process

import (
	"context"
	"errors"
	"github.com/keonho-kim/v-code/internal/tools/local/process/sandbox"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func Run(ctx context.Context, project, home, private, commandText string, confined bool) (output Output, err error) {
	var launch sandbox.Launch
	if confined {
		launch, err = sandbox.Prepare(project, private, commandText)
		if err != nil {
			return Output{Status: "scope_unavailable", ExitCode: -1}, err
		}
		defer func() { err = errors.Join(err, launch.Close()) }()
	}
	command := exec.CommandContext(ctx, "/bin/sh", "-c", commandText)
	if confined {
		command = exec.CommandContext(ctx, launch.Program, launch.Arguments...)
	}
	command.Dir = project
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "LANG=C.UTF-8"}
	if confined {
		command.Env = launch.Environment
	}
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
	output = Output{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: -1, Truncated: stdout.truncated || stderr.truncated, Status: "execution_failed"}
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
