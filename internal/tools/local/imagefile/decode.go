// Purpose: Decode bounded raster snapshots and normalize formats unsupported by vision APIs.
// Pattern: Pure codec adapter with cancellable stream boundaries.
// Usage: Local read_image execution supplies a stable file snapshot.
// Related: internal/agent/media/image.go, internal/tools/local/imagefile/webp.go.
package imagefile

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"

	"github.com/keonho-kim/meca/internal/agent/media"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Decode identifies the actual file format independently of its extension.
func Decode(ctx context.Context, path string, data []byte) (media.Image, error) {
	if err := ctx.Err(); err != nil {
		return media.Image{}, err
	}
	if len(data) == 0 || len(data) > media.MaxBytes {
		return media.Image{}, fmt.Errorf("image must contain 1 to %d bytes", media.MaxBytes)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return media.Image{}, fmt.Errorf("unsupported or invalid raster image (PNG, JPEG, WebP, GIF, BMP, TIFF): %w", err)
	}
	if config.Width < 1 || config.Height < 1 || config.Width > media.MaxPixels || config.Height > media.MaxPixels || int64(config.Width)*int64(config.Height) > media.MaxPixels {
		return media.Image{}, fmt.Errorf("image exceeds the %d-pixel decode budget", media.MaxPixels)
	}
	input := data
	animatedPNG := false
	if format == "png" {
		animatedPNG, err = pngAnimation(data)
		if err != nil {
			return media.Image{}, err
		}
	}
	animatedWebP := format == "webp" && len(data) > 20 && string(data[12:16]) == "VP8X" && data[20]&2 != 0
	if animatedWebP {
		input, err = firstWebPFrame(data)
		if err != nil {
			return media.Image{}, err
		}
		frame, _, err := image.DecodeConfig(bytes.NewReader(input))
		if err != nil || frame.Width < 1 || frame.Height < 1 || frame.Width > media.MaxPixels || frame.Height > media.MaxPixels || int64(frame.Width)*int64(frame.Height) > media.MaxPixels {
			return media.Image{}, fmt.Errorf("invalid or excessive image frame dimensions")
		}
	}
	decoded, _, err := image.Decode(contextReader{ctx, bytes.NewReader(input)})
	if err != nil {
		return media.Image{}, fmt.Errorf("decode image: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return media.Image{}, err
	}
	bounds := decoded.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 || int64(bounds.Dx())*int64(bounds.Dy()) > media.MaxPixels {
		return media.Image{}, fmt.Errorf("decoded image exceeds its pixel budget")
	}
	mime := map[string]string{"png": "image/png", "jpeg": "image/jpeg", "webp": "image/webp"}[format]
	// Animated/container formats become one static snapshot; ordinary compressed
	// PNG/JPEG/WebP bytes are reused without an additional quality or encoding pass.
	if mime == "" || animatedWebP || animatedPNG {
		var output bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.BestSpeed}
		if err = encoder.Encode(&boundedWriter{ctx: ctx, writer: &output}, decoded); err != nil {
			return media.Image{}, fmt.Errorf("encode image snapshot: %w", err)
		}
		data, mime = output.Bytes(), "image/png"
	}
	value, err := media.New(path, mime, format, bounds.Dx(), bounds.Dy(), data)
	value.Representative = animatedWebP || format == "gif" || format == "tiff" || animatedPNG
	return value, err
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

type boundedWriter struct {
	ctx    context.Context
	writer io.Writer
	size   int
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) > media.MaxBytes-w.size {
		return 0, fmt.Errorf("converted image exceeds %d bytes", media.MaxBytes)
	}
	n, err := w.writer.Write(p)
	w.size += n
	return n, err
}
