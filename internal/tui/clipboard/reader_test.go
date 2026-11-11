// Purpose: Verify clipboard raster decoding, private file reads and central error projection.
// Pattern: Deterministic input-boundary regression tests.
// Usage: go test ./internal/tui/clipboard.
// Related: internal/tui/clipboard/image.go, internal/tui/clipboard/file_image.go.
package clipboard

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func fixturePNG(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewGray(image.Rect(0, 0, 14, 9))); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestImageSnapshotAndLocalCopiedFile(t *testing.T) {
	data := fixturePNG(t)
	value, err := readImage(t.Context(), func(context.Context) ([]byte, error) { return data, nil })
	if err != nil || value.Width != 14 || value.Height != 9 {
		t.Fatal(value, err)
	}
	path := filepath.Join(t.TempDir(), "image with spaces.png")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	u := url.URL{Scheme: "file", Path: path}
	got, err := readClipboardFile(t.Context(), u.String())
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal(err)
	}
	for _, invalid := range []string{"https://example.com/a.png", "file://remote/a.png", "file:///a?secret=1"} {
		if _, err := readClipboardFile(t.Context(), invalid); err == nil {
			t.Fatal("accepted nonlocal URL", invalid)
		}
	}
}

func TestRemoteClipboardIsNeverRead(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "remote")
	if _, err := ReadImage(t.Context()); err == nil {
		t.Fatal("remote clipboard was accessed")
	}
	if handled, code := RunWorker([]string{"meca", workerArgument, strconv.Itoa(os.Getppid())}); !handled || code != workerFailed {
		t.Fatal("private worker bypassed remote guard", handled, code)
	}
}

func TestAlreadyCancelledReadAndFaultIdentity(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := readClipboardFile(ctx, "file:///no-file-access-after-cancel"); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled file read attempted I/O", err)
	}
	if _, err := readImage(ctx, func(context.Context) ([]byte, error) { return fixturePNG(t), nil }); err == nil {
		t.Fatal("cancellation ignored")
	}
	err := faults.New(diag.Unavailable, diag.ClipboardImageWSLUnavailable)
	if clipboardError(err) != err {
		t.Fatal("WSL failure identity replaced")
	}
}
