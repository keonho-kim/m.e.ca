// Purpose: Isolate synchronous macOS pasteboard calls in the current executable with bounded IPC.
// Pattern: Private executable worker and owned cancellable subprocess.
// Usage: main dispatches RunWorker before startup; TUI paste starts it only on macOS.
// Related: cmd/meca/main.go, internal/tui/clipboard/reader.go, internal/tui/clipboard/process.go.
package clipboard

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	native "golang.design/x/clipboard"
)

const workerArgument = "__meca_clipboard_image__"
const (
	workerEmpty  = 20
	workerLimit  = 21
	workerFailed = 22
)

// RunWorker handles only the private child invocation and must precede normal initialization.
func RunWorker(args []string) (bool, int) {
	if len(args) < 2 || args[1] != workerArgument {
		return false, 0
	}
	if len(args) != 3 {
		return true, workerFailed
	}
	parent, err := strconv.Atoi(args[2])
	if err != nil || parent != os.Getppid() || remoteTerminal() {
		return true, workerFailed
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	value, err := nativeSelection(ctx)
	if err != nil {
		status := byte(workerFailed)
		if errors.Is(err, native.ErrNoData) {
			status = workerEmpty
		}
		if errors.Is(err, native.ErrImageLimit) {
			status = workerLimit
		}
		var header [9]byte
		header[0] = status
		if _, err = os.Stdout.Write(header[:]); err != nil {
			return true, workerFailed
		}
		return true, 0
	}
	if err = writeFrame(os.Stdout, value); err != nil {
		return true, workerFailed
	}
	return true, 0
}

func readNativeWorker(ctx context.Context) (native.Image, error) {
	executable, err := os.Executable()
	if err != nil {
		return native.Image{}, err
	}
	cmd := exec.CommandContext(ctx, executable, workerArgument, strconv.Itoa(os.Getpid()))
	return readProcess(ctx, cmd)
}

func writeFrame(out io.Writer, value native.Image) error {
	data := value.Data
	var header [9]byte
	if value.FileURL != "" {
		header[0] = 1
		data = []byte(value.FileURL)
	}
	binary.BigEndian.PutUint64(header[1:], uint64(len(data)))
	if _, err := out.Write(header[:]); err != nil {
		return err
	}
	_, err := out.Write(data)
	return err
}
