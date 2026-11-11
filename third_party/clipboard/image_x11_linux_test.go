//go:build linux && !android

// Purpose: Reject blocking credential sources before reading the X11 authority file.
// Pattern: Isolated filesystem boundary regression test.
// Usage: Dependency tests on Linux; no display or personal clipboard is accessed.
// Related: third_party/clipboard/image_x11_wire_linux.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestImageX11RejectsAuthorityPipe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authority")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XAUTHORITY", path)
	done := make(chan struct{})
	go func() { imageCookie(0); close(done) }()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-done:
		return
	case <-timer.C:
		// Unblock the old reader before failing so the fixture leaves no worker.
		file, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			t.Fatal(err)
		}
		file.Close()
		<-done
		t.Fatal("X11 authority lookup blocked on a nonregular file")
	}
}
