//go:build linux || darwin

// Purpose: Own diagnostic stderr without changing inherited descriptor flags.
// Pattern: Unix output adapter.
// Usage: Process startup and emergency diagnostics attempt writes only when stderr is writable.
// Related: internal/diagnostics/logger.go, internal/diagnostics/emergency.go.
package diagnostics

import (
	"sync"

	"golang.org/x/sys/unix"
)

type Stderr struct {
	mu sync.Mutex
	fd int
}

func OpenStderr() (*Stderr, error) {
	fd, err := unix.Dup(2)
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(fd)
	return &Stderr{fd: fd}, nil
}

func (w *Stderr) Write(body []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.fd < 0 {
		return 0, unix.EBADF
	}
	for {
		// Dup shares its open-file flags with the parent and possibly stdout.
		// Never set O_NONBLOCK here. Readiness avoids waiting on a full pipe;
		// an actual kernel/device write still has the platform's I/O lifetime.
		poll := []unix.PollFd{{Fd: int32(w.fd), Events: unix.POLLOUT}}
		ready, err := unix.Poll(poll, 0)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, err
		}
		if ready == 0 {
			return 0, unix.EAGAIN
		}
		n, err := unix.Write(w.fd, body)
		if err == unix.EINTR {
			continue
		}
		return n, err
	}
}

func (w *Stderr) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.fd < 0 {
		return nil
	}
	fd := w.fd
	w.fd = -1
	return unix.Close(fd)
}

func emergencyWrite(body []byte) {
	output, err := OpenStderr()
	if err != nil {
		return
	}
	_, _ = output.Write(body)
	_ = output.Close()
}
