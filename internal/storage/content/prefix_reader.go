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

	"io"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/klauspost/compress/zstd"

	// TextPrefix never publishes a preview before validating the complete original
	// length, digest and JSON string. The encoded database body remains caller-owned.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (c *Codec) TextPrefix(ctx context.Context, codec, size int, id string, body []byte, limit int) (text string, clipped bool, err error) {
	return c.textPrefix(ctx, codec, size, id, body, limit, false)
}

func (c *Codec) RawTextPrefix(ctx context.Context, codec, size int, id string, body []byte, limit int) (string, bool, error) {
	return c.textPrefix(ctx, codec, size, id, body, limit, true)
}

func (c *Codec) textPrefix(ctx context.Context, codec, size int, id string, body []byte, limit int, raw bool) (text string, clipped bool, err error) {
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
		return "", false, faults.Errorf(diagcatalog.ContentInvalidStoredTextLengthOrPreviewLimit)
	}
	switch codec {
	case 0:
		if len(body) != size || ID(body) != id {
			return "", false, faults.Errorf(diagcatalog.ContentStoredPayloadIntegrityCheckFailed)
		}
		if raw {
			return rawTextPrefix(body, limit)
		}
		return StringPrefix(body, limit)
	case 1:
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return "", false, err
		}
		defer func() { err = errors.Join(err, reader.Close()) }()
		return readProjection(ctx, reader, size, id, limit, raw)
	case 2:
		if err := checkFrameLength(body, size); err != nil {
			return "", false, err
		}
		text, clipped, err = c.prefixZstd(ctx, body, size, id, limit, raw)
		if errors.Is(err, zstd.ErrWindowSizeExceeded) || errors.Is(err, zstd.ErrDecoderSizeExceeded) {
			// A wide history window can cost more memory than the full original.
			// Bulk decoding uses the already charged original-size bound instead.
			restored, restoreErr := c.Restore(codec, size, id, body)
			if restoreErr != nil {
				return "", false, restoreErr
			}
			if raw {
				return rawTextPrefix(restored, limit)
			}
			return StringPrefix(restored, limit)
		}
		return text, clipped, err
	default:
		return "", false, faults.Errorf(diagcatalog.ContentUnsupportedPayloadCodec, codec)
	}
}

func readProjection(ctx context.Context, source io.Reader, size int, id string, limit int, raw bool) (string, bool, error) {
	reader := io.LimitReader(source, int64(size)+1)
	digest := sha256.New()
	var parser interface {
		write([]byte) error
		finish(int) (string, bool, error)
	}
	if raw {
		parser = &rawPrefix{limit: limit}
	} else {
		value := newPrefixString(size, limit)
		parser = &value
	}
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
				return "", false, faults.Errorf(diagcatalog.ContentStoredPayloadLengthExceeded)
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
		return "", false, faults.Errorf(diagcatalog.ContentStoredPayloadIntegrityCheckFailed)
	}
	return parser.finish(limit)
}
