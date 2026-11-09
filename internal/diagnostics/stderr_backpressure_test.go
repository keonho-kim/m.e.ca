//go:build darwin || linux

// Purpose: Verify ordinary pipe backpressure cannot strand diagnostic shutdown.
// Pattern: Real Unix I/O ownership regression test.
// Usage: go test ./internal/diagnostics -run TestPartlyFullStderr.
// Related: internal/diagnostics/stderr_unix.go, internal/diagnostics/logger.go.
package diagnostics

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestPartlyFullStderrPipeRemainsClosable(t *testing.T) {
	fds := make([]int, 2)
	if err := unix.Pipe(fds); err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[0])
	defer unix.Close(fds[1])
	if err := unix.SetNonblock(fds[1], true); err != nil {
		t.Fatal(err)
	}
	body, buffered := make([]byte, maxRecordBytes-1), 0
	for {
		n, err := unix.Write(fds[1], body)
		buffered += max(0, n)
		if errors.Is(err, unix.EAGAIN) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := unix.Read(fds[0], make([]byte, 1024)); err != nil {
		t.Fatal(err)
	}
	if err := unix.SetNonblock(fds[1], false); err != nil {
		t.Fatal(err)
	}
	output, err := openStderr(fds[1])
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	finished := make(chan error, 1)
	go func() { _, err := output.Write(body); finished <- err }()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		_, _ = unix.Read(fds[0], make([]byte, buffered))
		<-finished
		t.Fatal("stderr write blocked behind ordinary pipe backpressure")
	}
	go func() { finished <- output.Close() }()
	select {
	case err := <-finished:
		if !errors.Is(err, ErrWrite) {
			t.Fatal("undelivered diagnostics did not report output failure", err)
		}
	case <-ctx.Done():
		_, _ = unix.Read(fds[0], make([]byte, buffered))
		<-finished
		t.Fatal("diagnostic relay was not terminated and joined")
	}
}
