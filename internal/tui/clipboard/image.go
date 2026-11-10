// Purpose: Read one local clipboard image into a bounded immutable raster snapshot.
// Pattern: Cancellable clipboard I/O adapter with private temporary ownership.
// Usage: The session's image-paste worker calls ReadImage; no terminal thread blocks on clipboard I/O.
// Related: internal/tui/clipboard/native.go, internal/media/imagefile/decode.go.
package clipboard

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/media/imagefile"
)

func ReadImage(ctx context.Context) (media.Image, error) {
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return media.Image{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ClipboardImageRemote)
	}
	return readImage(ctx, nativeImage)
}

func readImage(ctx context.Context, write func(context.Context, string) error) (result media.Image, cause error) {
	dir, err := os.MkdirTemp("", "meca-clipboard-")
	if err != nil {
		return media.Image{}, err
	}
	defer func() { cause = errors.Join(cause, os.RemoveAll(dir)) }()
	path := dir + "/image"
	if err := write(ctx, path); err != nil {
		return media.Image{}, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		return media.Image{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return media.Image{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ClipboardImageUnavailable)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return media.Image{}, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > media.MaxBytes {
		return media.Image{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageDataInvalid)
	}
	data := make([]byte, int(info.Size()))
	if _, err := io.ReadFull(file, data); err != nil {
		return media.Image{}, err
	}
	var extra [1]byte
	if _, err := file.Read(extra[:]); !errors.Is(err, io.EOF) {
		return media.Image{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageDataInvalid)
	}
	return imagefile.Decode(ctx, "clipboard-image", data)
}
