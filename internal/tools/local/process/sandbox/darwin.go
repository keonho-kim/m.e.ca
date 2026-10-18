//go:build darwin

// Purpose: Apply the shared read, write and secret policy to a macOS shell and its descendants.
// Pattern: macOS Seatbelt Adapter.
// Usage: Process.Run requests a launch after Permission approval; failures never fall back.
// Related: internal/tools/local/filesystem/boundary.go, internal/tools/local/process/sandbox/environment.go.
package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/keonho-kim/v-code/internal/tools/local/filesystem"
)

func Prepare(ctx context.Context, b *filesystem.Boundary, command string) (launch Launch, err error) {
	if err = b.CheckTrees(ctx); err != nil {
		return launch, err
	}
	if _, err = os.Stat("/usr/bin/sandbox-exec"); err != nil {
		return launch, fmt.Errorf("filesystem sandbox unavailable: %w", err)
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
	developer, err := developerRuntime(ctx, b.Project(), b.Home(), b.Private())
	if err != nil {
		return launch, err
	}
	if developer != "" {
		launch.Environment = append(launch.Environment, "DEVELOPER_DIR="+developer)
	}
	profile := `(version 1)
(deny default)
(allow process-fork sysctl-read)
(allow signal (target same-sandbox))
(allow file-read* process-exec file-map-executable)
(allow file-write-data (literal "/dev/null") (literal "/dev/zero"))
(allow network-outbound (remote tcp) (remote udp) (literal "/private/var/run/mDNSResponder"))
(allow network-bind network-inbound (local ip))
(deny file-link)
`
	for _, path := range append(b.Writes(), launch.Scratch) {
		profile += "(allow file-write* (subpath " + strconv.Quote(path) + "))\n"
		profile += "(deny file-write-unlink (literal " + strconv.Quote(path) + "))\n"
	}
	for _, path := range b.Protected() {
		for parent := filepath.Dir(path); parent != "/"; parent = filepath.Dir(parent) {
			for _, root := range b.Writes() {
				if filesystem.Within(root, parent) {
					profile += "(deny file-write-unlink (literal " + strconv.Quote(parent) + "))\n"
				}
			}
		}
		profile += "(deny file-read* file-write* process-exec file-map-executable (subpath " + strconv.Quote(path) + "))\n"
	}
	launch.Program = "/usr/bin/sandbox-exec"
	reader, writer, pipeErr := os.Pipe()
	if pipeErr != nil {
		return launch, pipeErr
	}
	launch.setup = reader
	launch.Files = append(launch.Files, writer)
	launch.Arguments = []string{"-p", profile, "/bin/sh", "-c", `printf 'ready\n' >&3 || exit 125; exec 3>&-; exec /bin/sh -c "$1"`, "v-shell", command}
	return launch, nil
}
