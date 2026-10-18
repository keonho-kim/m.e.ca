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
	"slices"
	"syscall"

	"github.com/keonho-kim/v-code/internal/tools/local/filesystem"
	"github.com/keonho-kim/v-code/internal/tools/local/process/sandbox/linux"
)

func Prepare(ctx context.Context, b *filesystem.Boundary, command string) (launch Launch, err error) {
	if err := b.CheckTrees(ctx); err != nil {
		return launch, err
	}
	if err := linux.Probe(); err != nil {
		return launch, fmt.Errorf("project sandbox unavailable: %w", err)
	}
	launch, err = scratch(b.Project())
	if err != nil {
		return launch, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, launch.Close())
			launch = Launch{}
		}
	}()
	request := linux.Request{Project: b.Project(), Private: b.Private(), Scratch: launch.Scratch, Command: command, Writable: b.Writes(), Protected: b.Protected()}
	request.Identities, err = b.RootIdentities()
	if err != nil {
		return launch, err
	}
	entries, err := os.ReadDir("/")
	if err != nil {
		return launch, err
	}
	for _, entry := range entries {
		switch entry.Name() {
		case "dev", "proc", "sys", "run":
			continue
		}
		request.Runtime = append(request.Runtime, filepath.Join("/", entry.Name()))
	}
	additional, err := linux.ReadRuntimeRoots(b.Home())
	if err != nil {
		return launch, err
	}
	for _, path := range additional {
		if !slices.Contains(request.Runtime, path) {
			request.Runtime = append(request.Runtime, path)
		}
	}
	// DNS symlinks may point into /run, whose host control sockets remain hidden.
	if path, e := filepath.EvalSymlinks("/etc/resolv.conf"); e == nil && filesystem.Within("/run", path) {
		request.Runtime = append(request.Runtime, path)
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
		Cloneflags:                 syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWIPC | syscall.CLONE_NEWUTS,
		UidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Geteuid(), Size: 1}},
		GidMappings:                []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getegid(), Size: 1}},
		GidMappingsEnableSetgroups: false,
		Pdeathsig:                  syscall.SIGKILL,
	}
	return launch, nil
}
