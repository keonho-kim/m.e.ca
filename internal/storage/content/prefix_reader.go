// Purpose: Verify complete compressed text with bounded scratch while returning only its visible prefix.
// Pattern: Cancellable streaming integrity boundary with caller-owned codec resources.
// Usage: SQLite history previews call TextPrefix after charging the original-size read budget.
// Related: internal/storage/content/prefix_string.go, internal/storage/content/prefix_zstd.go, internal/storage/content/compression.go.
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

// TextPrefix never publishes a preview before validating the complete original
// length, digest and JSON string. The encoded database body remains caller-owned.
func (c *Codec) TextPrefix(ctx context.Context, codec, size int, id string, body []byte, limit int) (text string, clipped bool, err error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	defer func() {
		if cause := ctx.Err(); cause != nil {
			text, clipped = "", false
			if !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}()
	if size < 1 || size > MaxBytes || limit < 1 {
		return "", false, fmt.Errorf("invalid stored text length or preview limit")
	}
	switch codec {
	case 0:
		if len(body) != size || ID(body) != id {
			return "", false, fmt.Errorf("stored payload integrity check failed")
		}
		return StringPrefix(body, limit)
	case 1:
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return "", false, err
		}
		defer func() { err = errors.Join(err, reader.Close()) }()
		return readPrefix(ctx, reader, size, id, limit)
	case 2:
		if err := checkFrameLength(body, size); err != nil {
			return "", false, err
		}
		text, clipped, err = c.prefixZstd(ctx, body, size, id, limit)
		if errors.Is(err, zstd.ErrWindowSizeExceeded) || errors.Is(err, zstd.ErrDecoderSizeExceeded) {
			// A wide history window can cost more memory than the full original.
			// Bulk decoding uses the already charged original-size bound instead.
			raw, restoreErr := c.Restore(codec, size, id, body)
			if restoreErr != nil {
				return "", false, restoreErr
			}
			return StringPrefix(raw, limit)
		}
		return text, clipped, err
	default:
		return "", false, fmt.Errorf("unsupported payload codec %d", codec)
	}
}

func readPrefix(ctx context.Context, source io.Reader, size int, id string, limit int) (string, bool, error) {
	reader := io.LimitReader(source, int64(size)+1)
	digest := sha256.New()
	parser := newPrefixString(size, limit)
	buffer := make([]byte, 32<<10)
	total := 0
	for {
		if err := ctx.Err(); err != nil {
			return "", false, err
		}
		n, err := reader.Read(buffer)
		if n > 0 {
			total += n
			if total > size {
				return "", false, fmt.Errorf("stored payload length exceeded")
			}
			digest.Write(buffer[:n])
			if parseErr := parser.write(buffer[:n]); parseErr != nil {
				return "", false, parseErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false, err
		}
	}
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	if total != size || hex.EncodeToString(digest.Sum(nil)) != id {
		return "", false, fmt.Errorf("stored payload integrity check failed")
	}
	return parser.finish(limit)
}
