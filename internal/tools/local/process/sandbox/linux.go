//go:build linux

// Purpose: Build a project filesystem namespace using the bundled Bubblewrap helper.
// Pattern: Linux process isolation adapter.
// Usage: Process.Run uses Prepare before invoking an approved project command.
// Related: internal/tools/local/process/sandbox/environment.go, internal/tools/local/process/sandbox/bundle/extract_linux.go.
package sandbox

import (
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/tools/local/process/sandbox/bundle"
	"os"
	"path/filepath"
)

func Prepare(project, private, command string) (Launch, error) {
	binary, err := bundle.Executable(private)
	if err != nil {
		return Launch{}, fmt.Errorf("project sandbox unavailable: %w", err)
	}
	launch, err := scratch(project)
	if err != nil {
		return Launch{}, err
	}
	args := []string{"--die-with-parent", "--new-session", "--unshare-user", "--unshare-pid", "--unshare-ipc", "--unshare-uts", "--unshare-net", "--cap-drop", "ALL"}
	// The new root starts empty. Only runtimes, the project and private scratch
	// are mounted; sibling projects, user state and inherited host sockets are absent.
	for _, path := range append(runtimePaths(), "/lib", "/lib64") {
		if _, err := os.Stat(path); err == nil {
			args = append(args, "--ro-bind", path, path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return Launch{}, errors.Join(err, launch.Close())
		}
	}
	args = append(args, "--proc", "/proc", "--dev", "/dev", "--bind", project, project, "--bind", launch.Scratch, launch.Scratch)
	relative, err := filepath.Rel(project, private)
	if err != nil {
		return Launch{}, errors.Join(err, launch.Close())
	}
	if filepath.IsLocal(relative) {
		// V state remains inaccessible even when the chosen project is its ancestor.
		args = append(args, "--tmpfs", private, "--remount-ro", private)
	}
	args = append(args, "--chdir", project, "--", "/bin/sh", "-c", command)
	launch.Program = binary
	launch.Arguments = args
	return launch, nil
}
