// Purpose: Read immutable encoded request segments without concatenating full conversation bodies.
// Pattern: Replayable I/O value with independent reader cursors.
// Usage: The HTTP adapter supplies a fresh Reader and exact Len for every attempt.
// Related: internal/provider/protocol/encoder.go, internal/provider/httpclient/request.go.
package protocol

import (
	"bytes"
	"io"
)

// Body retains immutable request fragments until their HTTP attempts finish.
// Each Reader has its own cursor, so retries cannot observe a consumed stream.
type Body struct {
	parts  [][]byte
	single []byte
	size   int
}

func (b Body) Len() int { return b.size }

func (b Body) Reader() io.Reader {
	if b.parts == nil {
		return bytes.NewReader(b.single)
	}
	return &bodyReader{parts: b.parts, size: b.size}
}

type bodyReader struct {
	parts  [][]byte
	index  int
	offset int
	size   int
}

func (r *bodyReader) Len() int          { return r.size }
func (r *bodyReader) Replay() io.Reader { return &bodyReader{parts: r.parts, size: r.size} }

func (r *bodyReader) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	n := 0
	for r.index < len(r.parts) {
		part := r.parts[r.index]
		copied := copy(dst[n:], part[r.offset:])
		n += copied
		r.offset += copied
		if r.offset == len(part) {
			r.index++
			r.offset = 0
		}
		if n == len(dst) {
			return n, nil
		}
	}
	if n > 0 {
		return n, nil
	}
	return 0, io.EOF
}
