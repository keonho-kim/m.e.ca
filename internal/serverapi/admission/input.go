// Purpose: Reserve actual input growth and release parsing ownership explicitly.
// Pattern: Request-owned byte reservation and reader adapter.
// Usage: HTTP bodies and WebSocket frames share a process input Budget.
// Related: internal/serverapi/transport.go, internal/serverapi/socket.go.
package admission

import (
	"context"
	"io"
	"sync"
)

type Input struct {
	mu     sync.Mutex
	budget *Budget
	bytes  int64
	closed bool
}

func NewInput(budget *Budget, initial int64) (*Input, error) {
	input := &Input{budget: budget}
	if initial > 0 {
		if err := input.reserve(initial); err != nil {
			return nil, err
		}
	}
	return input, nil
}

func (i *Input) reserve(bytes int64) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return ErrCapacity
	}
	if bytes <= i.bytes {
		return nil
	}
	// Charge in fixed chunks so one-byte reads do not create reservation
	// objects or acquire a new release closure for every byte.
	bytes = (bytes + (32 << 10) - 1) / (32 << 10) * (32 << 10)
	if !i.budget.resize(i.bytes, bytes) {
		return ErrCapacity
	}
	i.bytes = bytes
	return nil
}

func (i *Input) Resize(bytes int64) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed || bytes < 0 {
		return ErrCapacity
	}
	bytes = (bytes + (32 << 10) - 1) / (32 << 10) * (32 << 10)
	if !i.budget.resize(i.bytes, bytes) {
		return ErrCapacity
	}
	i.bytes = bytes
	return nil
}

func (i *Input) Release() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return
	}
	i.closed = true
	i.budget.resize(i.bytes, 0)
	i.bytes = 0
}

func (i *Input) Reader(reader io.Reader) io.Reader { return &inputReader{input: i, reader: reader} }

type inputReader struct {
	input  *Input
	reader io.Reader
	read   int64
}

func (r *inputReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if err := r.input.reserve(r.read + int64(len(p))); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	r.read += int64(n)
	return n, err
}

type inputKey struct{}

func WithInput(ctx context.Context, input *Input) context.Context {
	return context.WithValue(ctx, inputKey{}, input)
}

func ReleaseInput(ctx context.Context) {
	if input, ok := ctx.Value(inputKey{}).(*Input); ok {
		input.Release()
	}
}

func ResizeInput(ctx context.Context, bytes int64) error {
	if input, ok := ctx.Value(inputKey{}).(*Input); ok {
		return input.Resize(bytes)
	}
	return nil
}
