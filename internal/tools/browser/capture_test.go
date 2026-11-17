// Purpose: Verify stdout and stderr byte ceilings through the copy paths used by subprocess capture.
// Pattern: Regression tests for writer contracts, including optional io interfaces.
// Usage: go test ./internal/tools/browser -run TestCaptureByteLimit.
// Related: internal/tools/browser/capture.go.
package browser

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestCaptureByteLimit(t *testing.T) {
	const limit = 8192
	t.Run("accepted", func(t *testing.T) {
		body := strings.Repeat("x", limit)
		capture := &boundedBuffer{remaining: limit}
		if n, err := io.Copy(capture, struct{ io.Reader }{strings.NewReader(body)}); err != nil || n != limit || string(capture.Bytes()) != body || capture.limited {
			t.Fatal("bounded content changed", n, err)
		}
	})
	for _, kind := range []string{"reader", "file"} {
		t.Run(kind, func(t *testing.T) {
			body := strings.Repeat("x", limit*8)
			var source io.Reader = struct{ io.Reader }{strings.NewReader(body)}
			if kind == "file" {
				file, err := os.CreateTemp(t.TempDir(), "output")
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				if _, err := file.WriteString(body); err != nil {
					t.Fatal(err)
				}
				if _, err := file.Seek(0, io.SeekStart); err != nil {
					t.Fatal(err)
				}
				source = file
			}
			capture := &boundedBuffer{remaining: limit}
			_, err := io.Copy(capture, source)
			if err == nil || !capture.limited || len(capture.Bytes()) > limit {
				t.Fatalf("capture bypassed ceiling: error=%v, limited=%v, bytes=%d", err, capture.limited, len(capture.Bytes()))
			}
		})
	}
}
