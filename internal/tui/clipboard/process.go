// Purpose: Receive one size-prefixed clipboard snapshot and join its owned subprocess on every exit.
// Pattern: Bounded binary IPC adapter.
// Usage: macOS reader workers and WSL PowerShell use the same frame contract.
// Related: internal/tui/clipboard/worker.go, internal/tui/clipboard/wsl.go.
package clipboard

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	native "golang.design/x/clipboard"
)

func readProcess(ctx context.Context, cmd *exec.Cmd) (native.Image, error) {
	cmd.WaitDelay = time.Second
	// WSL forwards an interrupt through its interop channel. An immediate
	// SIGKILL can destroy only the proxy and strand the Windows process.
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return native.Image{}, err
	}
	defer pipe.Close()
	if err = cmd.Start(); err != nil {
		return native.Image{}, err
	}
	// WSL's host process can outlive its Linux proxy and retain stdout. Closing
	// our read end on cancellation makes frame reads independent of that proxy.
	stop := context.AfterFunc(ctx, func() { _ = pipe.Close() })
	defer stop()
	value, readErr := readFrame(pipe)
	var abort *time.Timer
	var aborted chan struct{}
	if readErr != nil && ctx.Err() == nil {
		_ = cmd.Cancel()
		aborted = make(chan struct{})
		abort = time.AfterFunc(cmd.WaitDelay, func() {
			_ = cmd.Process.Kill()
			close(aborted)
		})
	}
	waitErr := cmd.Wait()
	if abort != nil && !abort.Stop() {
		<-aborted
	}
	if ctx.Err() != nil {
		return native.Image{}, ctx.Err()
	}
	if errors.Is(readErr, native.ErrImageLimit) || errors.Is(readErr, native.ErrNoData) {
		return native.Image{}, readErr
	}
	if waitErr != nil {
		return native.Image{}, waitErr
	}
	return value, readErr
}

func readFrame(r io.Reader) (native.Image, error) {
	var header [9]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return native.Image{}, err
	}
	n := binary.BigEndian.Uint64(header[1:])
	if header[0] == workerEmpty && n == 0 {
		return native.Image{}, native.ErrNoData
	}
	if header[0] == workerLimit && n == 0 {
		return native.Image{}, native.ErrImageLimit
	}
	limit := uint64(media.MaxBytes)
	if header[0] == 1 {
		limit = 8192
	} else if header[0] != 0 {
		return native.Image{}, native.ErrUnsupported
	}
	if n == 0 {
		return native.Image{}, native.ErrNoData
	}
	if n > limit {
		return native.Image{}, native.ErrImageLimit
	}
	data := make([]byte, int(n))
	if _, err := io.ReadFull(r, data); err != nil {
		return native.Image{}, err
	}
	var extra [1]byte
	if n, err := r.Read(extra[:]); n != 0 || !errors.Is(err, io.EOF) {
		return native.Image{}, native.ErrUnsupported
	}
	if header[0] == 1 {
		return native.Image{FileURL: string(data)}, nil
	}
	return native.Image{Data: data}, nil
}
