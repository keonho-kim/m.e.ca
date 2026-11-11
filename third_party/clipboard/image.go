// Purpose: Define bounded read-only clipboard image values without decoding pixels.
// Pattern: Native I/O boundary shared by supported desktop platforms.
// Usage: MECA calls ReadImage; its image codec owns decoding and pixel budgets.
// Related: third_party/clipboard/image_linux.go, third_party/clipboard/image_darwin.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"context"
	"errors"
)

var ErrImageLimit = errors.New("clipboard image exceeds byte limit")

// Image owns one immutable raster snapshot or a local file URL, never both.
type Image struct {
	Data    []byte
	FileURL string
}

// ReadImage reads one image on demand without global initialization or watchers.
func ReadImage(ctx context.Context, maxBytes int) (Image, error) {
	if err := ctx.Err(); err != nil {
		return Image{}, err
	}
	if maxBytes <= 0 {
		return Image{}, ErrImageLimit
	}
	value, err := readImageBounded(ctx, maxBytes)
	if ctx.Err() != nil {
		return Image{}, ctx.Err()
	}
	return value, err
}
