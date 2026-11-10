// Purpose: Compress independent detail blocks for bounded disk storage and direct page reads.
// Pattern: Indexed chunk archive with a shared physical byte budget.
// Usage: documentSpool writes sequentially; documentCache decompresses only requested blocks.
// Related: internal/serverapi/detail/spool.go, internal/storage/content/compression.go.
package detail

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/content"
	"io"
	"os"
	"sync"
)

const detailBlock = 64 << 10

type spoolBudget struct {
	mu   sync.Mutex
	used int64
}
type chunkLocation struct {
	offset             int64
	bytes, codec, size int
	digest             string
}
type chunkArchive struct {
	file     *os.File
	codec    content.Codec
	buffer   []byte
	index    []chunkLocation
	physical int64
	budget   *spoolBudget
}

func (a *chunkArchive) Write(body []byte) (int, error) {
	total := len(body)
	for len(body) > 0 {
		n := min(detailBlock-len(a.buffer), len(body))
		a.buffer = append(a.buffer, body[:n]...)
		body = body[n:]
		if len(a.buffer) == detailBlock {
			if err := a.flush(); err != nil {
				return total - len(body), err
			}
		}
	}
	return total, nil
}
func (a *chunkArchive) flush() error {
	if len(a.buffer) == 0 {
		return nil
	}
	packed, err := a.codec.Encode(a.buffer)
	codec := 2
	if len(a.buffer) < 1024 || len(packed) >= len(a.buffer) {
		codec = 0
		packed = a.buffer
	}
	if err != nil {
		return err
	}
	if a.physical+int64(len(packed)) > maxSpoolBytes {
		return faults.New(diagcatalog.ResourceExhausted, diagcatalog.DetailSpoolBudget)
	}
	if a.budget != nil {
		a.budget.mu.Lock()
		if a.budget.used+int64(len(packed)) > maxSpoolBytes {
			a.budget.mu.Unlock()
			return faults.New(diagcatalog.ResourceExhausted, diagcatalog.DetailSpoolBudget)
		}
		a.budget.used += int64(len(packed))
		a.budget.mu.Unlock()
	}
	location := chunkLocation{offset: a.physical, bytes: len(packed), codec: codec, size: len(a.buffer), digest: content.ID(a.buffer)}
	a.physical += int64(len(packed))
	if n, err := a.file.Write(packed); err != nil {
		return err
	} else if n != len(packed) {
		return io.ErrShortWrite
	}
	a.index = append(a.index, location)
	a.buffer = a.buffer[:0]
	return nil
}
func (a *chunkArchive) ReadAt(body []byte, offset int64) (int, error) {
	written := 0
	for len(body) > 0 {
		block := int(offset / detailBlock)
		if block >= len(a.index) {
			return written, io.EOF
		}
		location := a.index[block]
		packed := make([]byte, location.bytes)
		if _, err := a.file.ReadAt(packed, location.offset); err != nil {
			return written, err
		}
		raw, err := a.codec.DecodeVerified(location.codec, location.size, location.digest, packed)
		if err != nil {
			return written, err
		}
		n := copy(body, raw[offset%detailBlock:])
		written += n
		offset += int64(n)
		body = body[n:]
	}
	return written, nil
}
func (a *chunkArchive) release() {
	_ = a.codec.Close()
	if a.budget != nil {
		a.budget.mu.Lock()
		a.budget.used -= a.physical
		a.budget.mu.Unlock()
	}
	a.physical = 0
	a.buffer = nil
	a.index = nil
}
