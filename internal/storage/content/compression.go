// Purpose: Reuse synchronous Zstandard resources independently of the persistence backend.
// Pattern: Caller-owned codec lifecycle with serialized access.
// Usage: A session store owns one Codec and calls Close after its users stop.
// Related: internal/storage/content/payload.go, internal/storage/sqlite/store.go.
package content

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/klauspost/compress/zstd"
)

type Codec struct {
	mu      sync.Mutex
	encoder *zstd.Encoder
	decoder *zstd.Decoder
	buffer  []byte
	closed  bool
}

func (c *Codec) Encode(raw []byte) ([]byte, error) {
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

func (c *Codec) decode(body []byte, size int) ([]byte, error) {
	var header zstd.Header
	if err := header.Decode(body); err != nil {
		return nil, err
	}
	if size < 1 || size > MaxBytes || !header.HasFCS || header.FrameContentSize != uint64(size) {
		return nil, fmt.Errorf("invalid stored payload frame length")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, fmt.Errorf("payload codec is closed")
	}
	if c.decoder == nil {
		decoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(MaxBytes), zstd.WithDecodeAllCapLimit(true))
		if err != nil {
			return nil, err
		}
		c.decoder = decoder
	}
	return c.decoder.DecodeAll(body, make([]byte, 0, size))
}

func (c *Codec) Close() error {
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
