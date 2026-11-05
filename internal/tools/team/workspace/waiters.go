// Purpose: Admit conflicting targets fairly and wake only affected workspace waiters.
// Pattern: Path-scoped cancellable admission with lazy notifications.
// Usage: Access reserve methods create a waiter only after encountering a real conflict.
// Related: internal/tools/team/workspace/workspace.go, internal/tools/team/workspace/reservation.go.
package workspace

import (
	"context"
	"slices"
)

type waiter struct {
	keys    []target
	command bool
	changed chan struct{}
}

func (w *Access) ready(keys []target, command bool, self *waiter) bool {
	if !command && w.files >= 16 {
		return false
	}
	for _, held := range w.writes {
		if conflicts(held, keys) {
			return false
		}
	}
	// An overlapping newcomer cannot overtake an earlier waiter. Unrelated
	// paths can still proceed immediately, even while another path is busy.
	for _, prior := range w.waiters {
		if prior == self {
			break
		}
		if conflicts(prior.keys, keys) {
			return false
		}
	}
	return true
}

// Caller holds mu. Notifications never perform filesystem I/O or allocate.
func (w *Access) notify(keys []target, fileSlot bool) {
	for _, pending := range w.waiters {
		if fileSlot && !pending.command || conflicts(keys, pending.keys) {
			select {
			case pending.changed <- struct{}{}:
			default:
			}
		}
	}
}

func (w *Access) remove(pending *waiter) {
	for i, entry := range w.waiters {
		if entry != pending {
			continue
		}
		copy(w.waiters[i:], w.waiters[i+1:])
		w.waiters[len(w.waiters)-1] = nil
		w.waiters = w.waiters[:len(w.waiters)-1]
		if len(w.waiters) == 0 {
			w.waiters = nil
		}
		w.notify(pending.keys, false)
		return
	}
}

func (w *Access) reserve(ctx context.Context, paths []string, command bool, obsolete <-chan struct{}, waiting func() error) (*Lease, error) {
	if err := admissionError(ctx, obsolete); err != nil {
		return nil, err
	}
	keys := make([]target, len(paths))
	var pending *waiter
	defer func() {
		if pending != nil {
			w.mu.Lock()
			w.remove(pending)
			w.mu.Unlock()
		}
	}()
	for {
		if err := admissionError(ctx, obsolete); err != nil {
			return nil, err
		}
		// Only a related release or cancellation causes another alias check.
		for i, path := range paths {
			keys[i] = canonical(path)
		}
		w.mu.Lock()
		if pending != nil {
			copy(pending.keys, keys)
		}
		if w.ready(keys, command, pending) {
			if err := admissionError(ctx, obsolete); err != nil {
				w.mu.Unlock()
				return nil, err
			}
			value := &Lease{workspace: w, command: command}
			w.writes[value] = keys
			if !command {
				w.files++
			}
			if pending != nil {
				w.remove(pending)
				pending = nil
			}
			w.mu.Unlock()
			return value, nil
		}
		fresh := pending == nil
		if fresh {
			pending = &waiter{keys: slices.Clone(keys), command: command, changed: make(chan struct{}, 1)}
			w.waiters = append(w.waiters, pending)
		}
		w.mu.Unlock()
		if fresh && waiting != nil {
			if err := waiting(); err != nil {
				return nil, err
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-obsolete:
			return nil, admissionError(ctx, obsolete)
		case <-pending.changed:
		}
	}
}
