//go:build linux || darwin

// Purpose: Keep inherited output flags intact and bound queued writes behind a full pipe.
// Pattern: Unix descriptor ownership regression tests.
// Usage: go test ./internal/diagnostics -run TestStderr.
// Related: internal/diagnostics/stderr_unix.go, internal/diagnostics/writer.go.
package diagnostics

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func TestStderrKeepsInheritedFlags(t *testing.T) {
	before, err := unix.FcntlInt(2, unix.F_GETFL, 0)
	if err != nil {
		t.Fatal(err)
	}
	output, err := OpenStderr()
	if err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := unix.FcntlInt(2, unix.F_GETFL, 0)
	if err != nil || before != after {
		t.Fatal("diagnostics changed shared output flags", before, after, err)
	}
}

func TestStderrBoundsFullPipe(t *testing.T) {
	var fds [2]int
	if err := unix.Pipe(fds[:]); err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[0])
	defer unix.Close(fds[1])
	if err := unix.SetNonblock(fds[1], true); err != nil {
		t.Fatal(err)
	}
	body := make([]byte, 4096)
	for {
		_, err := unix.Write(fds[1], body)
		if errors.Is(err, unix.EAGAIN) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := unix.SetNonblock(fds[1], false); err != nil {
		t.Fatal(err)
	}
	output, err := openStderr(fds[1])
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	for range 512 {
		if _, err := output.Write(body); errors.Is(err, unix.EAGAIN) {
			return
		} else if err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("blocked stderr exceeded the bounded relay buffer")
}
