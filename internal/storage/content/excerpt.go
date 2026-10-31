// Purpose: Read bounded plain-text excerpts while verifying the complete payload length and digest.
// Pattern: Streaming integrity boundary with caller-owned codec resources.
// Usage: Independent review reads observations; Restore remains the full JSON hydration boundary.
// Related: internal/storage/content/payload.go, internal/storage/content/compression.go.
package content

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

// Excerpt returns opaque source bytes, not a parsed JSON value. It consumes and
// authenticates the entire source while retaining at most limit excerpt bytes.
func (c *Codec) Excerpt(ctx context.Context, codec, size int, id string, body []byte, limit int) ([]byte, error) {
	if size < 1 || size > MaxBytes || limit < 1 || limit > MaxBytes {
		return nil, fmt.Errorf("invalid payload excerpt bounds")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, fmt.Errorf("payload codec is closed")
	}
	var reader io.Reader = bytes.NewReader(body)
	var closeReader func() error
	switch codec {
	case 0:
	case 1:
		r, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		reader, closeReader = r, r.Close
	case 2:
		var header zstd.Header
		if err := header.Decode(body); err != nil || !header.HasFCS || header.FrameContentSize != uint64(size) {
			return nil, fmt.Errorf("invalid stored payload frame length")
		}
		if c.decoder == nil {
			d, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(MaxBytes), zstd.WithDecodeAllCapLimit(true))
			if err != nil {
				return nil, err
			}
			c.decoder = d
		}
		if err := c.decoder.Reset(reader); err != nil {
			return nil, err
		}
		reader = c.decoder
		closeReader = func() error { return c.decoder.Reset(nil) }
	default:
		return nil, fmt.Errorf("unsupported payload codec %d", codec)
	}
	if c.excerptBuffer == nil {
		c.excerptBuffer = make([]byte, 64*1024)
	}
	hash := sha256.New()
	out := excerptWriter{ctx: ctx, prefix: make([]byte, 0, min(size, limit)), limit: limit}
	n, err := io.CopyBuffer(io.MultiWriter(hash, &out), io.LimitReader(reader, int64(size)+1), c.excerptBuffer)
	if closeReader != nil {
		err = errors.Join(err, closeReader())
	}
	if err != nil {
		return nil, err
	}
	if n != int64(size) || hex.EncodeToString(hash.Sum(nil)) != id {
		return nil, fmt.Errorf("stored payload integrity check failed")
	}
	return out.prefix, nil
}

type excerptWriter struct {
	ctx    context.Context
	prefix []byte
	limit  int
}

func (w *excerptWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	w.prefix = append(w.prefix, p[:min(len(p), w.limit-len(w.prefix))]...)
	return len(p), nil
}
