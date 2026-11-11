// Purpose: Verify exact framed reads, byte admission and owned process cancellation.
// Pattern: Fake executable I/O contracts without personal clipboard access.
// Usage: go test ./internal/tui/clipboard -run 'TestFrame|TestClipboardProcess'.
// Related: internal/tui/clipboard/process.go, internal/tui/clipboard/worker.go.
package clipboard

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	native "golang.design/x/clipboard"
)

func TestFrameRejectsExcessTruncationAndTrailingBytes(t *testing.T) {
	var good bytes.Buffer
	if err := writeFrame(&good, native.Image{Data: fixturePNG(t)}); err != nil {
		t.Fatal(err)
	}
	value, err := readFrame(bytes.NewReader(good.Bytes()))
	if err != nil || !bytes.Equal(value.Data, fixturePNG(t)) {
		t.Fatal(value, err)
	}
	large := make([]byte, 9)
	binary.BigEndian.PutUint64(large[1:], media.MaxBytes+1)
	if _, err := readFrame(bytes.NewReader(large)); !errors.Is(err, native.ErrImageLimit) {
		t.Fatal(err)
	}
	for _, data := range [][]byte{good.Bytes()[:len(good.Bytes())-1], append(bytes.Clone(good.Bytes()), 'x')} {
		if _, err := readFrame(bytes.NewReader(data)); err == nil {
			t.Fatal("invalid frame admitted")
		}
	}
}

func TestClipboardProcessHelper(t *testing.T) {
	mode := os.Getenv("MECA_TEST_CLIPBOARD_PROCESS")
	if mode == "" {
		return
	}
	switch mode {
	case "frame":
		_ = writeFrame(os.Stdout, native.Image{Data: []byte("pixels")})
		os.Exit(0)
	case "png":
		_ = writeFrame(os.Stdout, native.Image{Data: fixturePNG(t)})
		os.Exit(0)
	case "empty":
		_, _ = os.Stdout.Write(makeStatusFrame(workerEmpty))
		os.Exit(0)
	case "limit":
		var h [9]byte
		binary.BigEndian.PutUint64(h[1:], media.MaxBytes+1)
		_, _ = os.Stdout.Write(h[:])
		select {}
	case "blocked":
		interrupts := make(chan os.Signal, 1)
		signal.Notify(interrupts, os.Interrupt)
		_, _ = os.Stderr.Write([]byte{'R'})
		<-interrupts
		signal.Stop(interrupts)
		_, _ = os.Stderr.Write([]byte{'C'})
		os.Exit(0)
	case "ignore":
		signal.Ignore(os.Interrupt)
		_, _ = os.Stderr.Write([]byte{'R'})
		time.Sleep(time.Hour) // Simulate a wedged child until the owner's forced termination.
	}
	os.Exit(1)
}

func helperCommand(t *testing.T, ctx context.Context, mode string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestClipboardProcessHelper$")
	cmd.Env = append(os.Environ(), "MECA_TEST_CLIPBOARD_PROCESS="+mode)
	return cmd
}

func TestClipboardProcessFramesAndSizeFailure(t *testing.T) {
	value, err := readProcess(t.Context(), helperCommand(t, t.Context(), "frame"))
	if err != nil || string(value.Data) != "pixels" {
		t.Fatal(value, err)
	}
	if _, err := readProcess(t.Context(), helperCommand(t, t.Context(), "empty")); !errors.Is(err, native.ErrNoData) {
		t.Fatal(err)
	}
	if _, err := readProcess(t.Context(), helperCommand(t, t.Context(), "limit")); !errors.Is(err, native.ErrImageLimit) {
		t.Fatal(err)
	}
}

func TestClipboardProcessCancellationJoins(t *testing.T) {
	t.Run("interrupt", func(t *testing.T) { verifyProcessCancellation(t, "blocked", true) })
	t.Run("force", func(t *testing.T) { verifyProcessCancellation(t, "ignore", false) })
}

func verifyProcessCancellation(t *testing.T, mode string, graceful bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cmd := helperCommand(t, ctx, mode)
	ready := make(chan struct{})
	stopped := make(chan struct{})
	cmd.Stderr = &readyWriter{ready: ready, stopped: stopped}
	done := make(chan error, 1)
	go func() {
		_, err := readProcess(ctx, cmd)
		done <- err
	}()
	defer cancel()
	select {
	case <-ready:
	case err := <-done:
		t.Fatal("helper ended before readiness", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled process read did not return cancellation", err)
	}
	if cmd.ProcessState == nil {
		t.Fatal("subprocess was not joined")
	}
	if graceful {
		select {
		case <-stopped:
		default:
			t.Fatal("helper was killed without receiving the interrupt")
		}
	} else if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); !ok || status.Signal() != syscall.SIGKILL {
		t.Fatal("unresponsive helper was not forcibly terminated", cmd.ProcessState)
	}
}

func makeStatusFrame(status byte) []byte {
	header := make([]byte, 9)
	header[0] = status
	return header
}

type readyWriter struct{ ready, stopped chan struct{} }

func (w *readyWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		ch := w.ready
		if b == 'C' {
			ch = w.stopped
		}
		select {
		case <-ch:
		default:
			close(ch)
		}
	}
	return len(p), nil
}
