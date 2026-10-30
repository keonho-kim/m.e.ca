// Purpose: Preserve static PNG bytes and recognize animation only from an actual control chunk.
// Pattern: Raster container regression tests.
// Usage: Image codec tests exercise read_image normalization independently of file extensions.
// Related: internal/tools/local/imagefile/decode.go, internal/tools/local/imagefile/png.go.
package imagefile

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"testing"
)

func pngChunk(kind string, body []byte) []byte {
	chunk := make([]byte, 12+len(body))
	binary.BigEndian.PutUint32(chunk, uint32(len(body)))
	copy(chunk[4:8], kind)
	copy(chunk[8:], body)
	binary.BigEndian.PutUint32(chunk[8+len(body):], crc32.ChecksumIEEE(chunk[4:8+len(body)]))
	return chunk
}

func TestPNGAnimationDetection(t *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	original := encoded.Bytes()
	for _, tc := range []struct {
		name, kind string
		body       []byte
		animated   bool
	}{
		{"plain", "", nil, false},
		{"text mentions animation marker", "tEXt", []byte("Comment\x00ordinary acTL text"), false},
		{"animation control", "acTL", []byte{0, 0, 0, 2, 0, 0, 0, 0}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := bytes.Clone(original[:33]) // The control chunk precedes IDAT, after IHDR.
			if tc.kind != "" {
				data = append(data, pngChunk(tc.kind, tc.body)...)
			}
			data = append(data, original[33:]...)
			got, err := Decode(context.Background(), "shot.png", data)
			if err != nil {
				t.Fatal(err)
			}
			if got.Representative != tc.animated || bytes.Equal(got.Data, data) == tc.animated {
				t.Fatalf("representative=%v unchanged=%v", got.Representative, bytes.Equal(got.Data, data))
			}
		})
	}
}
