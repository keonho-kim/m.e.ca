// Purpose: Bound Zstandard streaming history across every frame while sharing the store's decoder.
// Pattern: Scoped decoder options under the codec mutex.
// Usage: TextPrefix tries bounded streaming, then uses charged bulk restoration for wider frames.
// Related: internal/storage/content/prefix_reader.go, internal/storage/content/compression.go.
package content

import (
	"bytes"
	"context"
	"errors"

	"github.com/klauspost/compress/zstd"
)

func (c *Codec) prefixZstd(ctx context.Context, body []byte, size int, id string, limit int) (text string, clipped bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	decoder, err := c.decoderLocked()
	if err != nil {
		return "", false, err
	}
	// MaxMemory also caps single-segment windows. The cap applies to later
	// concatenated frames before their history buffers can be allocated.
	defer func() {
		err = errors.Join(err, decoder.ResetWithOptions(nil, zstd.WithDecoderMaxMemory(MaxBytes), zstd.WithDecoderMaxWindow(MaxBytes)))
	}()
	if err := decoder.ResetWithOptions(bytes.NewReader(body), zstd.WithDecoderMaxMemory(payloadWindowBytes), zstd.WithDecoderMaxWindow(payloadWindowBytes)); err != nil {
		return "", false, err
	}
	return readPrefix(ctx, decoder, size, id, limit)
}
