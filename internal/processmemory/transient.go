// Purpose: Admit cancellable byte reservations for transient decoding and payload transformations.
// Pattern: Process-owned byte admission with caller-owned release.
// Usage: Runtime, image decoders and SQLite reserve before materializing transient values.
// Related: internal/processmemory/reply.go, internal/tools/local/imagefile/decode.go, internal/storage/sqlite/commit.go.
package processmemory

import (
	"context"
	"sync"
)

const DecodeBytes int64 = 256 << 20
const PayloadBytes int64 = 512 << 20

type transientBudget struct {
	mu            sync.Mutex
	used, maximum int64
	changed       chan struct{}
}

var decodes = transientBudget{maximum: DecodeBytes}
var payloads = transientBudget{maximum: PayloadBytes}

func (b *transientBudget) wake() {
	if b.changed != nil {
		close(b.changed)
		b.changed = nil
	}
}
func (b *transientBudget) acquire(ctx context.Context, bytes int64) (func(), error) {
	bytes = min(b.maximum, max(int64(1), bytes))
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		b.mu.Lock()
		if bytes <= b.maximum-b.used {
			b.used += bytes
			b.mu.Unlock()
			return sync.OnceFunc(func() { b.mu.Lock(); b.used -= bytes; b.wake(); b.mu.Unlock() }), nil
		}
		if b.changed == nil {
			b.changed = make(chan struct{})
		}
		changed := b.changed
		b.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

func AcquireDecode(ctx context.Context, bytes int64) (func(), error) {
	return decodes.acquire(ctx, bytes)
}
func AcquirePayload(ctx context.Context, bytes int64) (func(), error) {
	return payloads.acquire(ctx, bytes)
}
