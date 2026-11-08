// Purpose: Read stream lines within the provider response budget and release oversized scratch buffers.
// Pattern: Response-owned buffered reader.
// Usage: ReadResponse consumes each borrowed line before advancing to the next one.
// Related: internal/provider/protocol/stream.go, internal/provider/protocol/chunk.go.
package protocol

import (
	"bufio"
	"context"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

const retainedFrameBytes = 64 * 1024

type streamLines struct {
	*bufio.Reader
	long  []byte
	limit int
}

func (r *streamLines) next(ctx context.Context) ([]byte, error) {
	if cap(r.long) > retainedFrameBytes {
		r.long = nil
	}
	r.long = r.long[:0]
	for {
		if err := context.Cause(ctx); err != nil {
			return nil, err
		}
		part, err := r.ReadSlice('\n')
		if r.limit > 0 && len(r.long)+len(part) > r.limit {
			return nil, &modelapi.OutputLimitError{Reason: "model stream frame byte limit"}
		}
		if len(r.long) == 0 && err != bufio.ErrBufferFull {
			return part, err
		}
		r.long = append(r.long, part...)
		if err != bufio.ErrBufferFull {
			return r.long, err
		}
	}
}

// skipBoundary never refills the reader: the preceding borrowed line must
// remain valid until its JSON has been consumed synchronously.
func (r *streamLines) skipBoundary() bool {
	if r.Buffered() == 0 {
		return false
	}
	peek, _ := r.Peek(min(2, r.Buffered()))
	n := 0
	if peek[0] == '\n' {
		n = 1
	} else if len(peek) == 2 && peek[0] == '\r' && peek[1] == '\n' {
		n = 2
	}
	if n == 0 {
		return false
	}
	_, err := r.Discard(n)
	return err == nil
}
