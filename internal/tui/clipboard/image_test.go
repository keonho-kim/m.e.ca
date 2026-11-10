// Purpose: Verify private clipboard-file cleanup and raster decoding without accessing personal clipboard data.
// Pattern: Fake command I/O boundary tests.
// Usage: go test ./internal/tui/clipboard.
// Related: internal/tui/clipboard/image.go.
package clipboard

import (
	"context"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestClipboardImageDecodeAndCleanup(t *testing.T) {
	var path string
	value, err := readImage(t.Context(), func(_ context.Context, target string) error {
		path = target
		file, err := os.Create(target)
		if err != nil {
			return err
		}
		defer file.Close()
		return png.Encode(file, image.NewGray(image.Rect(0, 0, 14, 9)))
	})
	if err != nil || value.Width != 14 || value.Height != 9 || len(value.Data) == 0 {
		t.Fatal(value, err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatal("clipboard spool retained", err)
	}
}

func TestMacImageClipboardScriptCompiles(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS compiler is unavailable on this platform")
	}
	command := exec.CommandContext(t.Context(), "/usr/bin/osacompile", "-o", filepath.Join(t.TempDir(), "image.scpt"), "-e", macImageScript)
	if body, err := command.CombinedOutput(); err != nil {
		t.Fatalf("AppKit clipboard script did not compile: %v %s", err, body)
	}
}

func TestRemoteClipboardIsNeverRead(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "remote")
	if _, err := ReadImage(t.Context()); err == nil {
		t.Fatal("remote clipboard was accessed")
	}
}
