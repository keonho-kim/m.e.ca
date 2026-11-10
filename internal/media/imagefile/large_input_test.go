// Purpose: Accept a compressed snapshot above the former 32MiB ceiling without copying unchanged image data.
// Pattern: Decoder and snapshot ownership boundary test.
// Usage: go test ./internal/media/imagefile -run TestLargeImageSnapshot.
// Related: internal/media/imagefile/decode.go, internal/agent/media/image.go.
package imagefile

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

func TestLargeImageSnapshotReusesCompressedBytes(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, image.NewGray(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}
	// A small JPEG with trailing padding isolates the compressed-byte ceiling
	// without allocating a large raster in the test.
	data := make([]byte, 33<<20)
	copy(data, encoded.Bytes())
	value, err := Decode(t.Context(), "large.jpg", data)
	if err != nil || value.Width != 1 || len(value.Data) != len(data) {
		t.Fatal("larger compressed snapshot rejected", err)
	}
	if &value.Data[0] != &data[0] {
		t.Fatal("unchanged compressed data was copied")
	}
}
