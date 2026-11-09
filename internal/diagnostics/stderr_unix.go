//go:build linux || darwin

// Purpose: Isolate inherited stderr writes behind an owned nonblocking pipe and joined relay.
// Pattern: Unix output adapter with a process I/O boundary.
// Usage: CLI and server logging start one lazy relay; Close drains or terminates it.
// Related: internal/diagnostics/stderr_relay.go, internal/diagnostics/logger.go, cmd/meca/main.go.
package diagnostics

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"golang.org/x/sys/unix"
)

type Stderr struct {
	mu       sync.Mutex
	fd       int
	command  *exec.Cmd
	wait     chan error
	failure  chan struct{}
	once     sync.Once
	closeErr error
}

var emergencyOutput atomic.Pointer[Stderr]

func OpenStderr() (*Stderr, error) {
	output, err := openStderr(2)
	if err == nil {
		emergencyOutput.CompareAndSwap(nil, output)
	}
	return output, err
}

func openStderr(source int) (*Stderr, error) {
	fd, err := unix.FcntlInt(uintptr(source), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return &Stderr{fd: fd, failure: make(chan struct{})}, nil
}

func (w *Stderr) Failure() <-chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.failure == nil {
		w.failure = make(chan struct{})
	}
	return w.failure
}

// startRelay requires w.mu. The child receives only the stream descriptors and
// its parent's PID; it gets no provider credentials or inherited environment.
func (w *Stderr) startRelay() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	var pipe [2]int
	syscall.ForkLock.RLock()
	err = unix.Pipe(pipe[:])
	if err == nil {
		for _, fd := range pipe {
			unix.CloseOnExec(fd)
		}
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return err
	}
	if err := unix.SetNonblock(pipe[1], true); err != nil {
		return errors.Join(err, unix.Close(pipe[0]), unix.Close(pipe[1]))
	}
	input := os.NewFile(uintptr(pipe[0]), "diagnostic-relay-input")
	output := os.NewFile(uintptr(w.fd), "diagnostic-stderr")
	command := exec.Command(executable, stderrRelayArgument, strconv.Itoa(os.Getpid()))
	command.Env = []string{"GOMAXPROCS=1"}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Stdin, command.Stdout = input, output
	if err := command.Start(); err != nil {
		_ = output.Close()
		w.fd = -1
		return errors.Join(err, input.Close(), unix.Close(pipe[1]))
	}
	w.fd, w.command, w.wait = pipe[1], command, make(chan error, 1)
	if w.failure == nil {
		w.failure = make(chan struct{})
	}
	wait, failure := w.wait, w.failure
	go func() {
		err := command.Wait()
		if err != nil {
			close(failure)
		}
		wait <- err
	}()
	return errors.Join(input.Close(), output.Close())
}

func (w *Stderr) Write(body []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.fd < 0 {
		return 0, unix.EBADF
	}
	if len(body) == 0 {
		return 0, nil
	}
	if w.command == nil {
		if err := w.startRelay(); err != nil {
			return 0, err
		}
	}
	for {
		n, err := unix.Write(w.fd, body)
		if err == unix.EINTR {
			continue
		}
		return max(0, n), err
	}
}

func (w *Stderr) Close() error {
	w.once.Do(func() {
		emergencyOutput.CompareAndSwap(w, nil)
		w.mu.Lock()
		if w.fd >= 0 {
			w.closeErr = unix.Close(w.fd)
			w.fd = -1
		}
		command, wait := w.command, w.wait
		w.mu.Unlock()
		if command == nil {
			return
		}
		timer := time.NewTimer(catalog.LogDrainTimeout)
		defer timer.Stop()
		select {
		case err := <-wait:
			if err != nil {
				w.closeErr = errors.Join(w.closeErr, ErrWrite)
			}
		case <-timer.C:
			_ = command.Process.Kill()
			<-wait
			w.closeErr = errors.Join(w.closeErr, ErrWrite)
		}
	})
	return w.closeErr
}

func emergencyWrite(body []byte) {
	if output := emergencyOutput.Load(); output != nil {
		_, _ = output.Write(body)
	}
}
