// Purpose: Read one local clipboard image into a bounded immutable raster snapshot.
// Pattern: Cancellable clipboard adapter with request-owned bytes.
// Usage: The session's image-paste worker calls ReadImage; no terminal thread blocks on clipboard I/O.
// Related: internal/tui/clipboard/reader.go, internal/media/imagefile/decode.go.
package clipboard

import (
	"context"
	"os"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/media/imagefile"
)

func ReadImage(ctx context.Context) (media.Image, error) {
	if remoteTerminal() {
		return media.Image{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ClipboardImageRemote)
	}
	return readImage(ctx, readLocalImage)
}

func remoteTerminal() bool {
	return os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != ""
}

func readImage(ctx context.Context, read func(context.Context) ([]byte, error)) (media.Image, error) {
	data, err := read(ctx)
	if err != nil {
		return media.Image{}, err
	}
	return imagefile.Decode(ctx, "clipboard-image", data)
}
