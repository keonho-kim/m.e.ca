// Purpose: Snapshot one explicitly copied local image file with bounded stable-file reads.
// Pattern: Local file URL trust boundary.
// Usage: Native clipboard readers return a file URL when the clipboard holds a copied file.
// Related: internal/tui/clipboard/reader.go, internal/media/imagefile/decode.go.
package clipboard

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"syscall"

	"github.com/keonho-kim/meca/internal/agent/media"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func readClipboardFile(ctx context.Context, location string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	u, err := url.Parse(location)
	if err != nil || u.Scheme != "file" || (u.Host != "" && u.Host != "localhost") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || len(u.Path) > 4096 {
		return nil, faults.New(diag.InvalidArgument, diag.ClipboardImageUnavailable)
	}
	f, err := os.OpenFile(u.Path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, faults.New(diag.InvalidArgument, diag.ClipboardImageUnavailable)
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() <= 0 {
		return nil, faults.New(diag.InvalidArgument, diag.ClipboardImageUnavailable)
	}
	if before.Size() > media.MaxBytes {
		return nil, faults.New(diag.InvalidArgument, diag.ClipboardImageTooLarge)
	}
	data := make([]byte, int(before.Size()))
	for offset := 0; offset < len(data); {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		n, err := f.Read(data[offset:min(offset+65536, len(data))])
		offset += n
		if errors.Is(err, io.EOF) {
			return nil, faults.New(diag.Conflict, diag.ClipboardImageChanged)
		}
		if err != nil {
			return nil, err
		}
	}
	var extra [1]byte
	if n, err := f.Read(extra[:]); n != 0 || !errors.Is(err, io.EOF) {
		return nil, faults.New(diag.Conflict, diag.ClipboardImageChanged)
	}
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, faults.New(diag.Conflict, diag.ClipboardImageChanged)
	}
	return data, ctx.Err()
}
