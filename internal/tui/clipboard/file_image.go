// Purpose: Copy an explicitly selected local clipboard file URL into the private image spool.
// Pattern: Bounded regular-file I/O boundary.
// Usage: AppKit file clipboard export calls copyClipboardFile; raster decoding remains in image.go.
// Related: internal/tui/clipboard/native.go, internal/tui/clipboard/image.go.
package clipboard

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"syscall"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func copyClipboardFile(ctx context.Context, location, target string) (cause error) {
	u, err := url.Parse(location)
	if err != nil || u.Scheme != "file" || u.Host != "" && u.Host != "localhost" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || len(u.Path) > 4096 {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ClipboardImageUnavailable)
	}
	source, err := os.OpenFile(u.Path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ClipboardImageUnavailable)
	}
	defer func() { cause = errors.Join(cause, source.Close()) }()
	info, err := source.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > media.MaxBytes {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageDataInvalid)
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { cause = errors.Join(cause, file.Close()) }()
	buffer := make([]byte, 64<<10)
	writer := imageWriter{writer: file}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := source.Read(buffer)
		if _, writeErr := writer.Write(buffer[:n]); writeErr != nil {
			return writeErr
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}
	after, err := source.Stat()
	if err != nil {
		return err
	}
	if writer.bytes != int(info.Size()) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageDataInvalid)
	}
	return ctx.Err()
}
