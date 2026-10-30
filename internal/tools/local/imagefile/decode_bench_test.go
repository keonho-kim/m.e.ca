// Purpose: Measure decode allocation and conversion cost for ordinary screenshot-sized images.
// Pattern: Codec microbenchmark.
// Usage: go test ./internal/tools/local/imagefile -run '^$' -bench BenchmarkDecode -benchmem.
// Related: internal/tools/local/imagefile/decode.go.
package imagefile

import (
	"bytes"
	"context"
	"golang.org/x/image/bmp"
	"image"
	"image/png"
	"testing"
)

func BenchmarkDecode(b *testing.B) {
	picture := image.NewNRGBA(image.Rect(0, 0, 1024, 768))
	for _, format := range []string{"png", "bmp"} {
		b.Run(format, func(b *testing.B) {
			var data bytes.Buffer
			var err error
			if format == "png" {
				err = png.Encode(&data, picture)
			} else {
				err = bmp.Encode(&data, picture)
			}
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Decode(context.Background(), "shot."+format, data.Bytes()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
