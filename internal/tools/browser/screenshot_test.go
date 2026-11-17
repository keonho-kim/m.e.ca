// Purpose: Reject oversized screenshot pixels before allocating or decoding the full image.
// Pattern: PNG header and exact-file-length contract tests.
// Usage: go test ./internal/tools/browser -run TestViewportPixels.
// Related: internal/tools/browser/screenshot.go.
package browser

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestViewportPixels(t *testing.T) {
	var source bytes.Buffer
	if err := png.Encode(&source, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	original := source.Bytes()
	for _, huge := range []bool{false, true} {
		data := bytes.Clone(original)
		if huge {
			binary.BigEndian.PutUint32(data[16:20], 4096)
			binary.BigEndian.PutUint32(data[20:24], 4096)
			binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
		}
		path := filepath.Join(t.TempDir(), "viewport.png")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := readViewport(file, int64(len(data)))
		file.Close()
		if huge {
			if err == nil || got != nil {
				t.Fatal("oversized pixels admitted")
			}
		} else if err != nil || !bytes.Equal(got, data) {
			t.Fatal("valid PNG changed", err)
		}
	}
}
