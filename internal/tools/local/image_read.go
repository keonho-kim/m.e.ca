// Purpose: Read one bounded image snapshot with a buffer sized from its opened file.
// Pattern: Cancellable read boundary with explicit stale-source detection.
// Usage: viewImage supplies a regular file and its current size before codec validation.
// Related: internal/tools/local/image.go, internal/agent/media/image.go.
package local

import (
	"context"

	"io"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func readImageBytes(ctx context.Context, reader io.Reader, size int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if size < 0 || size > media.MaxBytes {
		return nil, faults.Errorf(diagcatalog.LocalImageExceedsBytes, media.MaxBytes)
	}
	// A single extra byte detects growth without expanding the original buffer.
	data := make([]byte, int(size)+1)
	n := 0
	for n < len(data) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		count, err := reader.Read(data[n:])
		n += count
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, io.ErrNoProgress
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if int64(n) != size {
		return nil, &faults.Failure{Code: diagcatalog.StaleTarget, Detail: diagcatalog.Text(diagcatalog.LocalImageSizeChangedWhileReadingItsSnapshot), Next: diagcatalog.Text(diagcatalog.LocalViewTheIntendedImageAgain)}
	}
	return data[:n:n], nil
}
