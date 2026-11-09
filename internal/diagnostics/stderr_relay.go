// Purpose: Forward diagnostics in an isolated process that exits when its parent disappears.
// Pattern: Private executable worker entrypoint.
// Usage: main and executable test entrypoints dispatch this mode before normal startup.
// Related: internal/diagnostics/stderr_unix.go, cmd/meca/main.go.
package diagnostics

import (
	"io"
	"os"
	"strconv"
	"time"
)

const stderrRelayArgument = "__meca_diagnostic_relay__"

// RunStderrRelay returns whether this is the private worker invocation and its
// exit code. The executable must exit immediately when handled is true.
func RunStderrRelay(args []string) (handled bool, code int) {
	if len(args) < 2 || args[1] != stderrRelayArgument {
		return false, 0
	}
	if len(args) != 3 {
		return true, 1
	}
	parent, err := strconv.Atoi(args[2])
	if err != nil || parent <= 0 || os.Getppid() != parent {
		return true, 1
	}
	done := make(chan error, 1)
	go func() { _, err := io.Copy(os.Stdout, os.Stdin); done <- err }()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if err != nil {
				return true, 1
			}
			return true, 0
		case <-ticker.C:
			if os.Getppid() != parent {
				// Process exit ends its blocked kernel write. The owner normally
				// kills and joins us; this also handles an abruptly killed owner.
				return true, 1
			}
		}
	}
}
