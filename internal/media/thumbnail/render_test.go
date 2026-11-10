// Purpose: Verify bounded image dimensions, exact original ownership and cancellation.
// Pattern: Deterministic codec contract test.
// Usage: go test ./internal/media/thumbnail.
// Related: internal/media/thumbnail/render.go.
package thumbnail

import (
	"bytes"
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/media"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestRenderBoundsAndCancellation(t *testing.T) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewNRGBA(image.Rect(0, 0, 1024, 512))); err != nil {
		t.Fatal(err)
	}
	source := bytes.Clone(buffer.Bytes())
	value, err := media.New("input.png", "image/png", "png", 1024, 512, source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Render(t.Context(), value)
	if err != nil {
		t.Fatal(err)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(result))
	if err != nil || format != "png" || config.Width != 256 || config.Height != 128 {
		t.Fatal(config, format, err)
	}
	if !bytes.Equal(source, buffer.Bytes()) {
		t.Fatal("thumbnail changed original bytes")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Render(ctx, value); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestRenderPreservesCameraOrientation(t *testing.T) {
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, image.NewNRGBA(image.Rect(0, 0, 4, 2)), nil); err != nil {
		t.Fatal(err)
	}
	tiff := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	data := append([]byte{}, buffer.Bytes()[:2]...)
	data = append(data, 0xff, 0xe1, 0, byte(len(payload)+2))
	data = append(data, payload...)
	data = append(data, buffer.Bytes()[2:]...)
	value, err := media.New("camera.jpg", "image/jpeg", "jpeg", 4, 2, data)
	if err != nil {
		t.Fatal(err)
	}
	body, err := Render(t.Context(), value)
	if err != nil {
		t.Fatal(err)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || config.Width != 2 || config.Height != 4 {
		t.Fatal("camera orientation lost", config, err)
	}
}
