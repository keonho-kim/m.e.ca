// Purpose: Bound simultaneous server work by reserved bytes or operation count.
// Pattern: Cancellable admission with explicit caller-owned release.
// Usage: Transport parsers and original-result writers share process-scoped budgets.
// Related: internal/serverapi/transport.go, internal/serverapi/openai/result.go.
package admission

import (
	"context"

	"sync"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var ErrCapacity = faults.Errorf(diagcatalog.AdmissionServerWorkExceedsItsAdmissionBudget)

type Budget struct {
	mu             sync.Mutex
	capacity, used int64
	wake           chan struct{}
}

func New(capacity int64) *Budget { return &Budget{capacity: capacity} }

func (b *Budget) resize(previous, next int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if next-previous > b.capacity-b.used {
		return false
	}
	b.used += next - previous
	if next < previous && b.wake != nil {
		close(b.wake)
		b.wake = nil
	}
	return true
}

func (b *Budget) take(weight int64) func() {
	b.used += weight
	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			b.used -= weight
			if b.wake != nil {
				close(b.wake)
				b.wake = nil
			}
		})
	}
}

func (b *Budget) Try(weight int64) (func(), bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if weight < 1 || weight > b.capacity-b.used {
		return nil, false
	}
	return b.take(weight), true
}

func (b *Budget) Acquire(ctx context.Context, weight int64) (func(), error) {
	if weight < 1 || weight > b.capacity {
		return nil, ErrCapacity
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		b.mu.Lock()
		if weight <= b.capacity-b.used {
			release := b.take(weight)
			b.mu.Unlock()
			return release, nil
		}
		if b.wake == nil {
			b.wake = make(chan struct{})
		}
		wake := b.wake
		b.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-wake:
		}
	}
}
