//go:build linux

// Purpose: Prepare a self-executing Linux sandbox with immutable input and a separate setup error channel.
// Pattern: Linux process launch adapter.
// Usage: Process.Run executes the launch after Scope and user approval checks.
// Related: internal/tools/local/process/sandbox/linux/child_linux.go, internal/tools/local/process/sandbox/environment.go.
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/keonho-kim/v-code/internal/tools/local/process/sandbox/linux"
)

func Prepare(ctx context.Context, project, home, private, command string) (launch Launch, err error) {
	if err := ctx.Err(); err != nil {
		return Launch{}, err
	}

	if err := linux.Probe(); err != nil {
		return launch, fmt.Errorf("project sandbox unavailable: %w", err)
	}
	project, err = filepath.EvalSymlinks(project)
	if err != nil {
		return launch, err
	}
	private, err = filepath.EvalSymlinks(private)
	if err != nil {
		return launch, err
	}
	launch, err = scratch(project)
	if err != nil {
		return launch, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, launch.Close())
			launch = Launch{}
		}
	}()
	request := linux.Request{Project: project, Private: private, Scratch: launch.Scratch, Command: command}
	request.UserRuntime, err = userRuntime(&launch, home, private)
	if err != nil {
		return launch, err
	}
	for _, path := range append(runtimePaths(), "/lib", "/lib64") {
		if _, statErr := os.Stat(path); statErr == nil {
			request.Runtime = append(request.Runtime, path)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return launch, statErr
		}
	}
	input, err := linux.Encode(request)
	if err != nil {
		return launch, err
	}
	launch.Files = append(launch.Files, input)
	reader, writer, err := os.Pipe()
	if err != nil {
		return launch, err
	}
	launch.setup = reader
	launch.Files = append(launch.Files, writer)
	// The executable pathname can be atomically replaced while V is running.
	// Re-execute the current image, never a replacement supplied by the project.
	launch.Program = "/proc/self/exe"
	launch.Name = linux.ChildName
	launch.Attributes = &syscall.SysProcAttr{
		Setpgid:                    true,
		Cloneflags:                 syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNET,
		UidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Geteuid(), Size: 1}},
		GidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getegid(), Size: 1}},
		GidMappingsEnableSetgroups: false,
		Pdeathsig:                  syscall.SIGKILL,
	}
	return launch, nil
}
