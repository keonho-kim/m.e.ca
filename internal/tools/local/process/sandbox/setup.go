// Purpose: Own the parent ends of sandbox startup descriptors and report isolation failures separately from command output.
// Pattern: Child process handshake.
// Usage: The process adapter closes launch inputs after Start and reads setup status after Wait.
// Related: internal/tools/local/process/sandbox/environment.go, internal/tools/local/process/run_unix.go.
package sandbox

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func (l Launch) CloseInputs() (err error) {
	for _, file := range l.Files {
		if closeErr := file.Close(); !errors.Is(closeErr, os.ErrClosed) {
			err = errors.Join(err, closeErr)
		}
	}
	return err
}

func (l Launch) SetupError() error {
	if l.setup == nil {
		return nil
	}
	message, err := io.ReadAll(io.LimitReader(l.setup, 2049))
	if err != nil {
		return fmt.Errorf("read sandbox setup status: %w", err)
	}
	if len(message) == 0 {
		return fmt.Errorf("project sandbox stopped before completing isolation")
	}
	if string(message) != "ready\n" {
		return fmt.Errorf("project sandbox unavailable: %s", message)
	}
	return nil
}
