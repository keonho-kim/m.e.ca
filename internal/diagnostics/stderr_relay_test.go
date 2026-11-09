//go:build linux || darwin

// Purpose: Verify relay delivery, inherited descriptor flags and asynchronous output failure.
// Pattern: Real Unix stream lifecycle tests.
// Usage: go test -race ./internal/diagnostics -run TestStderrRelay.
// Related: internal/diagnostics/stderr_unix.go, internal/diagnostics/writer.go.
package diagnostics

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestStderrRelayPreservesBytesAndFlags(t *testing.T) {
	var fds [2]int
	if err := unix.Pipe(fds[:]); err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fds[1])
	if err := unix.SetNonblock(fds[0], true); err != nil {
		t.Fatal(err)
	}
	input := os.NewFile(uintptr(fds[0]), "diagnostic-test-input")
	defer input.Close()
	if err := input.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := unix.SetNonblock(fds[1], true); err != nil {
		t.Fatal(err)
	}
	before, _ := unix.FcntlInt(uintptr(fds[1]), unix.F_GETFL, 0)
	output, err := openStderr(fds[1])
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	body := []byte(strings.Repeat("diagnostic 한글\n", 80))
	if n, err := output.Write(body); err != nil || n != len(body) {
		t.Fatal("relay did not accept the record", n, err)
	}
	after, err := unix.FcntlInt(uintptr(fds[1]), unix.F_GETFL, 0)
	if err != nil || after != before {
		t.Fatal("relay changed the inherited file flags", before, after, err)
	}
	got := make([]byte, len(body))
	if _, err := io.ReadFull(input, got); err != nil || !bytes.Equal(got, body) {
		t.Fatal("relay changed diagnostic bytes", err)
	}
	if err := output.Close(); err != nil {
		t.Fatal("delivered relay did not drain and join", err)
	}
}

func TestStderrRelayFailureReachesSink(t *testing.T) {
	var fds [2]int
	if err := unix.Pipe(fds[:]); err != nil {
		t.Fatal(err)
	}
	_ = unix.Close(fds[0])
	defer unix.Close(fds[1])
	output, err := openStderr(fds[1])
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	sink, err := New(Options{Writer: output, Redact: func(s string) string { return s }, Interrupt: output.Close})
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()
	sink.logger().Info("test diagnostic")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case <-sink.Failure():
	case <-ctx.Done():
		t.Fatal("relay output failure was hidden until another write")
	}
	if sink.Healthy() {
		t.Fatal("failed relay remained healthy")
	}
}
