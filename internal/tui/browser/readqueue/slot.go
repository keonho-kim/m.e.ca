// Purpose: Admit one active read and replace a single pending request on the event loop.
// Pattern: Per-feature bulkhead with latest-request backpressure.
// Usage: Directory and preview workers release their slot through completion notices, including cancelled reads.
// Related: internal/tui/session/directory.go, internal/tui/session/file_preview.go.
package readqueue

import "context"

type Slot[T any] struct {
	running  context.CancelFunc
	revision int
	pending  *T
}

func (s *Slot[T]) Submit(request T) bool {
	if s.running == nil {
		return true
	}
	s.running()
	s.pending = &request
	return false
}

func (s *Slot[T]) Stop() {
	if s.running != nil {
		s.running()
	}
	s.pending = nil
}

func (s *Slot[T]) Complete(revision int) *T {
	if s.running == nil || revision != s.revision {
		return nil
	}
	s.running()
	s.running = nil
	next := s.pending
	s.pending = nil
	return next
}

func (s *Slot[T]) Begin(revision int, cancel context.CancelFunc) {
	s.running, s.revision = cancel, revision
}
func (s *Slot[T]) Running() bool { return s.running != nil }
