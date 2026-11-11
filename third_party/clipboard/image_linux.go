//go:build linux && !android

// Purpose: Select a reachable native Linux image backend without cached initialization failures.
// Pattern: Request-scoped native adapter selection.
// Usage: ReadImage calls this with the caller's deadline and compressed-byte budget.
// Related: third_party/clipboard/image_x11_linux.go, third_party/clipboard/image_wayland_linux.go, third_party/clipboard/LICENSE.
package clipboard

import (
	"context"
	"errors"
	"os"
	"strings"
)

var imageTypes = []string{"text/uri-list", "image/png", "image/jpeg", "image/webp", "image/tiff", "image/bmp", "image/gif"}

func readImageBounded(ctx context.Context, limit int) (Image, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		value, err := imageWayland(ctx, limit)
		if !errors.Is(err, ErrUnavailable) && !errors.Is(err, ErrUnsupported) {
			return value, err
		}
	}
	if ctx.Err() != nil {
		return Image{}, ctx.Err()
	}
	return imageX11(ctx, limit)
}

func imageValue(mime string, data []byte) (Image, error) {
	if len(data) == 0 {
		return Image{}, ErrNoData
	}
	if mime != "text/uri-list" {
		return Image{Data: data}, nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return Image{FileURL: line}, nil
		}
	}
	return Image{}, ErrNoData
}
