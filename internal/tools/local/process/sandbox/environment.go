// Purpose: Describe an isolated shell launch and own its descriptors and temporary working files.
// Pattern: Process launch adapter contract.
// Usage: Process.Run executes the returned command and closes its scratch directory.
// Related: internal/tools/local/process/sandbox/darwin.go, internal/tools/local/process/sandbox/linux.go.
package sandbox

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

type Launch struct {
	Program                string
	Arguments, Environment []string
	Directory, Scratch     string
	Name                   string
	Attributes             *syscall.SysProcAttr
	Files                  []*os.File
	setup                  *os.File
}

func (l Launch) Close() error {
	err := l.CloseInputs()
	if l.setup != nil {
		if closeErr := l.setup.Close(); !errors.Is(closeErr, os.ErrClosed) {
			err = errors.Join(err, closeErr)
		}
	}
	if l.Scratch == "" {
		return err
	}
	return errors.Join(err, os.RemoveAll(l.Scratch))
}
func scratch(project string) (Launch, error) {
	directory, err := os.MkdirTemp("", "v-command-")
	if err != nil {
		return Launch{}, err
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return Launch{}, errors.Join(err, os.RemoveAll(directory))
	}
	return Launch{Directory: project, Scratch: resolved, Environment: []string{"PATH=" + os.Getenv("PATH"), "HOME=" + project, "TMPDIR=" + resolved, "TMP=" + resolved, "TEMP=" + resolved, "XDG_CACHE_HOME=" + resolved, "GOCACHE=" + filepath.Join(resolved, "go-cache"), "LANG=C.UTF-8"}}, nil
}

// Installed runtime resources are explicit read-only exceptions to project data scope.
func runtimePaths() []string {
	return []string{"/bin", "/sbin", "/usr/bin", "/usr/sbin", "/usr/lib", "/usr/lib64", "/usr/share", "/usr/include", "/usr/local/bin", "/usr/local/lib", "/usr/local/include", "/usr/local/share", "/usr/local/go"}
}
