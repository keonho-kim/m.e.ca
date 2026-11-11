// Purpose: Verify real native Linux and simulated Windows clipboard paths against an explicit fixture.
// Pattern: Opt-in isolated display and subprocess integration tests.
// Usage: Set MECA_CLIPBOARD_FIXTURE only inside the private Linux display test container.
// Related: internal/tui/clipboard/reader.go, internal/tui/clipboard/process.go.
package clipboard

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	native "golang.design/x/clipboard"
)

func TestFixtureNativeClipboard(t *testing.T) {
	path := os.Getenv("MECA_CLIPBOARD_FIXTURE")
	if path == "" || runtime.GOOS != "linux" {
		t.Skip("isolated Linux clipboard fixture required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	value, err := ReadImage(ctx)
	if err != nil || !bytes.Equal(value.Data, data) {
		t.Fatal("native raster snapshot mismatch", err)
	}
	if len(data) > 1024 {
		if _, err := native.ReadImage(ctx, 1024); !errors.Is(err, native.ErrImageLimit) {
			t.Fatal("native input limit not enforced", err)
		}
	}
}

func TestFixtureWSLClipboard(t *testing.T) {
	if os.Getenv("MECA_CLIPBOARD_FIXTURE") == "" || runtime.GOOS != "linux" {
		t.Skip("isolated Linux fixture required")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := "#!/bin/sh\n[ \"$1\" = -NoProfile ] || exit 1\n[ \"$2\" = -NonInteractive ] || exit 1\n[ \"$3\" = -STA ] || exit 1\nexec \"$MECA_TEST_CLIPBOARD_EXE\" -test.run=^TestClipboardProcessHelper$\n"
	if err = os.WriteFile(filepath.Join(dir, "powershell.exe"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("WSL_DISTRO_NAME", "IsolatedFixture")
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("MECA_TEST_CLIPBOARD_EXE", exe)
	t.Setenv("MECA_TEST_CLIPBOARD_PROCESS", "png")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	value, err := ReadImage(ctx)
	if err != nil || !bytes.Equal(value.Data, fixturePNG(t)) {
		t.Fatal("WSL raster bridge mismatch", err)
	}
}
