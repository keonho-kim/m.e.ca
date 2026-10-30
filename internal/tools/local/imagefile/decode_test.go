// Purpose: Verify common raster formats are decoded by content and rejected before excessive allocation.
// Pattern: Image codec contract tests.
// Usage: go test ./internal/tools/local/imagefile.
// Related: internal/tools/local/imagefile/decode.go.
package imagefile

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

func TestDecodeFormatsAndInvalidInputs(t *testing.T) {
	picture := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	picture.Set(2, 3, color.NRGBA{R: 255, A: 255})
	for name, encode := range map[string]func(*bytes.Buffer) error{
		"png":  func(b *bytes.Buffer) error { return png.Encode(b, picture) },
		"jpeg": func(b *bytes.Buffer) error { return jpeg.Encode(b, picture, nil) },
		"gif":  func(b *bytes.Buffer) error { return gif.Encode(b, picture, nil) },
		"bmp":  func(b *bytes.Buffer) error { return bmp.Encode(b, picture) },
		"tiff": func(b *bytes.Buffer) error { return tiff.Encode(b, picture, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			if err := encode(&b); err != nil {
				t.Fatal(err)
			}
			got, err := Decode(context.Background(), "misleading.extension", b.Bytes())
			if err != nil || got.Width != 8 || got.Height != 6 || got.Format != name {
				t.Fatalf("image=%+v err=%v", got, err)
			}
			if err := got.Check(); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, data := range [][]byte{nil, []byte("not an image"), {137, 80, 78, 71, 13, 10, 26, 10}} {
		if _, err := Decode(context.Background(), "bad.png", data); err == nil {
			t.Fatal("invalid image accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Decode(ctx, "a.png", []byte("bad")); err != context.Canceled {
		t.Fatalf("cancellation lost: %v", err)
	}
}
