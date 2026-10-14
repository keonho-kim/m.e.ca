//go:build darwin

// Purpose: Confine project shell filesystem access with the system Seatbelt launcher.
// Pattern: macOS process isolation adapter.
// Usage: Process.Run requests a launch; unsupported isolation fails without an unconfined retry.
// Related: internal/tools/local/process/sandbox/environment.go, internal/tools/local/process/sandbox/darwin_test.go.
package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Prepare(project, private, command string) (Launch, error) {
	if _, err := os.Stat("/usr/bin/sandbox-exec"); err != nil {
		return Launch{}, fmt.Errorf("project sandbox unavailable: %w", err)
	}
	launch, err := scratch(project)
	if err != nil {
		return Launch{}, err
	}
	profile := `(version 1)
(deny default)
(allow process-fork sysctl-read)
(allow signal (target same-sandbox))
(allow file-read-data (literal "/") (literal "/dev/null") (literal "/dev/random") (literal "/dev/urandom"))
(allow file-write-data (literal "/dev/null"))
`
	// Runtime and toolchain resources are read-only exceptions, not writable scope.
	for _, path := range append(runtimePaths(), "/opt/homebrew/bin", "/opt/homebrew/lib", "/opt/homebrew/opt", "/opt/homebrew/Cellar", "/opt/homebrew/include", "/opt/homebrew/share", "/System/Library", "/Library/Apple", "/private/var/db/dyld", project, launch.Scratch) {
		profile += "(allow file-read* process-exec file-map-executable (subpath " + strconv.Quote(path) + "))\n"
		for parent := filepath.Dir(path); ; parent = filepath.Dir(parent) {
			profile += "(allow file-read-metadata (literal " + strconv.Quote(parent) + "))\n"
			if parent == "/" {
				break
			}
		}
	}
	for _, path := range []string{project, launch.Scratch} {
		profile += "(allow file-write* (subpath " + strconv.Quote(path) + "))\n"
	}
	profile += "(deny file-read* file-write* (subpath " + strconv.Quote(private) + "))\n"
	if strings.ContainsRune(profile, 0) {
		return Launch{}, errors.Join(fmt.Errorf("invalid sandbox path"), launch.Close())
	}
	launch.Program = "/usr/bin/sandbox-exec"
	launch.Arguments = []string{"-p", profile, "/bin/sh", "-c", command}
	return launch, nil
}
