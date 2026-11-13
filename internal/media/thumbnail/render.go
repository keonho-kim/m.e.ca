// Purpose: Derive a small lossless display image under the shared pixel-decoding budget.
// Pattern: Cancellable codec boundary.
// Usage: The authenticated image endpoint loads originals only on thumbnail cache misses.
// Related: internal/serverapi/thumbnail.go, internal/processmemory/transient.go.
package thumbnail

import (
	"bytes"
	"context"
	"github.com/keonho-kim/meca/internal/agent/media"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/media/imagefile"
	"github.com/keonho-kim/meca/internal/processmemory"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	"image/png"
)

// Edge is the maximum display-preview dimension advertised by capabilities.
const Edge = 256

func Render(ctx context.Context, value media.Image) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(value.Data))
	if err != nil {
		return nil, err
	}
	pixels := int64(config.Width) * int64(config.Height)
	if config.Width < 1 || config.Height < 1 || config.Width > media.MaxPixels || config.Height > media.MaxPixels || pixels > media.MaxPixels {
		return nil, fault.Errorf(diag.MediaImageExceedsThePixelDecodeBudget, media.MaxPixels)
	}
	release, err := processmemory.AcquireDecode(ctx, pixels*8+int64(len(value.Data))*2+2<<20)
	if err != nil {
		return nil, err
	}
	defer release()
	input := value.Data
	if format == "webp" && len(input) > 20 && string(input[12:16]) == "VP8X" && input[20]&2 != 0 {
		input, err = imagefile.FirstWebPFrame(input, config.Width, config.Height)
		if err != nil {
			return nil, err
		}
	}
	source, _, err := image.Decode(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	width, height := source.Bounds().Dx(), source.Bounds().Dy()
	if width > Edge || height > Edge {
		if width >= height {
			height = max(1, height*Edge/width)
			width = Edge
		} else {
			width = max(1, width*Edge/height)
			height = Edge
		}
	}
	result := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(result, result.Bounds(), source, source.Bounds(), draw.Src, nil)
	result = orientPreview(result, displayOrientation(value.Data))
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&output, result); err != nil {
		return nil, err
	}
	return output.Bytes(), ctx.Err()
}
