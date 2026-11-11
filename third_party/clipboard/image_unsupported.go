//go:build !darwin && !linux

// Purpose: Reject the MECA read-only extension on platforms outside its distribution.
// Pattern: Build-selected native boundary.
// Usage: Build-selected rejection of image reads outside the distributed target set.
// Related: third_party/clipboard/image.go, third_party/clipboard/LICENSE.
package clipboard

import "context"

func readImageBounded(context.Context, int) (Image, error) { return Image{}, ErrUnsupported }
