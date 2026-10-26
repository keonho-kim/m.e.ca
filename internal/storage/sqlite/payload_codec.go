// Purpose: Reuse one synchronous Zstandard encoder and decoder per session store.
// Pattern: Store-owned resource with serialized codec access.
// Usage: Payload persistence uses encode/decode; Store.Close releases retained workspaces.
// Related: internal/storage/sqlite/payload.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/klauspost/compress/zstd"
)

type payloadCodec struct {
	mu      sync.Mutex
	encoder *zstd.Encoder
	decoder *zstd.Decoder
	buffer  []byte
	closed  bool
}

func (c *payloadCodec) encode(raw []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, fmt.Errorf("payload codec is closed")
	}
	if c.encoder == nil {
		encoder, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), zstd.WithWindowSize(128*1024), zstd.WithEncoderLevel(zstd.SpeedFastest))
		if err != nil {
			return nil, err
		}
		c.encoder = encoder
	}
	compressed := c.encoder.EncodeAll(raw, c.buffer[:0])
	if cap(compressed) <= 256*1024 {
		c.buffer = compressed[:0]
		return bytes.Clone(compressed), nil
	}
	return compressed, nil
}

func (c *payloadCodec) decode(body []byte, size int) ([]byte, error) {
	var header zstd.Header
	if err := header.Decode(body); err != nil {
		return nil, err
	}
	if size < 1 || size > maxEventBytes || !header.HasFCS || header.FrameContentSize != uint64(size) {
		return nil, fmt.Errorf("invalid stored payload frame length")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, fmt.Errorf("payload codec is closed")
	}
	if c.decoder == nil {
		decoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(maxEventBytes), zstd.WithDecodeAllCapLimit(true))
		if err != nil {
			return nil, err
		}
		c.decoder = decoder
	}
	return c.decoder.DecodeAll(body, make([]byte, 0, size))
}

func (c *payloadCodec) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	c.buffer = nil
	var err error
	if c.encoder != nil {
		err = c.encoder.Close()
		c.encoder = nil
	}
	if c.decoder != nil {
		c.decoder.Close()
		c.decoder = nil
	}
	return err
}
