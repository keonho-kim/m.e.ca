// Purpose: Verify streamed text with a request-owned decoder that never holds a codec lock across I/O.
// Pattern: Scoped streaming integrity and memory-admission boundary.
// Usage: SQLite visible previews lend chunked payload readers to StreamTextPrefix.
// Related: internal/storage/content/prefix_reader.go, internal/storage/sqlite/text_prefix.go.
package content

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"io"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/klauspost/compress/zstd"
)

const PreviewScratchBytes = 4*payloadWindowBytes + 64<<10

func StreamTextPrefix(ctx context.Context, codec, size int, id string, reader io.Reader, limit int, raw bool, reserve func(int) error) (text string, clipped bool, err error) {
	if size < 1 || size > MaxBytes || limit < 1 {
		return "", false, faults.Errorf(diagcatalog.ContentInvalidStoredTextLengthOrPreviewLimit)
	}
	switch codec {
	case 0:
		return readProjection(ctx, reader, size, id, limit, raw)
	case 1:
		decoded, err := gzip.NewReader(reader)
		if err != nil {
			return "", false, err
		}
		defer func() { err = errors.Join(err, decoded.Close()) }()
		return readProjection(ctx, decoded, size, id, limit, raw)
	case 2:
		buffered := bufio.NewReaderSize(reader, 32<<10)
		prefix, peekErr := buffered.Peek(zstd.HeaderMaxSize)
		if peekErr != nil && !errors.Is(peekErr, io.EOF) {
			return "", false, peekErr
		}
		var header zstd.Header
		if err := header.Decode(prefix); err != nil {
			return "", false, err
		}
		if !header.HasFCS || header.FrameContentSize != uint64(size) {
			return "", false, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadFrameLength)
		}
		window := header.WindowSize
		if header.SingleSegment {
			window = header.FrameContentSize
		}
		window = max(uint64(payloadWindowBytes), window)
		if window > MaxBytes {
			return "", false, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadFrameLength)
		}
		if reserve != nil {
			if err := reserve(int(3*window) + PreviewScratchBytes); err != nil {
				return "", false, err
			}
		}
		decoded, err := zstd.NewReader(buffered, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true), zstd.WithDecoderMaxMemory(window), zstd.WithDecoderMaxWindow(window))
		if err != nil {
			return "", false, err
		}
		defer decoded.Close()
		return readProjection(ctx, decoded, size, id, limit, raw)
	default:
		return "", false, faults.Errorf(diagcatalog.ContentUnsupportedPayloadCodec, codec)
	}
}
