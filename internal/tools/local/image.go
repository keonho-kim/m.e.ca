// Purpose: Read a stable policy-checked raster snapshot without recording a filesystem mutation.
// Pattern: Read-only filesystem adapter with one decoder at a time per executor.
// Usage: Prepared read_image operations call readImage after path revalidation.
// Related: internal/tools/local/image_read.go, internal/tools/local/imagefile/decode.go, internal/tools/local/paths.go.
package local

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/keonho-kim/meca/internal/agent/media"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/imagefile"
)

func (op *operation) readImage(ctx context.Context) (out toolapi.Result, err error) {
	select {
	case op.executor.imageReads <- struct{}{}:
		defer func() { <-op.executor.imageReads }()
	case <-ctx.Done():
		return out, ctx.Err()
	}
	info, err := op.source.root.Stat(op.source.relative)
	if err != nil {
		return out, err
	}
	if !info.Mode().IsRegular() {
		return out, fmt.Errorf("image must be a regular file")
	}
	file, err := op.source.root.OpenFile(op.source.relative, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return out, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	before, err := file.Stat()
	if err != nil {
		return out, err
	}
	if !before.Mode().IsRegular() {
		return out, fmt.Errorf("image must be a regular file")
	}
	data, err := readImageBytes(ctx, file, before.Size())
	if err != nil {
		return out, err
	}
	current, err := op.executor.resolve(op.settings.Scope, op.input.Path)
	if err != nil {
		return out, err
	}
	after, err := current.root.Stat(current.relative)
	if err != nil {
		return out, err
	}
	if current.absolute != op.source.absolute || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return out, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "image changed while reading its snapshot", Next: "Read the intended image again."}
	}
	image, err := imagefile.Decode(ctx, op.executor.redact(op.source.absolute), data)
	if err != nil {
		return out, err
	}
	// Unchanged transport bytes already have a digest. Rehash the source only
	// when a container/frame conversion produced different compressed bytes.
	sourceDigest := image.Digest
	if len(data) != len(image.Data) || &data[0] != &image.Data[0] {
		digest := sha256.Sum256(data)
		sourceDigest = fmt.Sprintf("%x", digest)
	}
	metadata := struct {
		Path, Format, MIME       string
		Width, Height, SizeBytes int
		Representative           bool
	}{image.Path, image.Format, image.MIME, image.Width, image.Height, len(image.Data), image.Representative}
	body, err := json.Marshal(metadata)
	if err != nil {
		return out, err
	}
	return toolapi.Result{Images: []media.Image{image}, Content: body, Summary: "Image loaded", Versions: map[string]string{op.source.absolute: sourceDigest}}, nil
}
