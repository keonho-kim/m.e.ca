// Purpose: Charge context hydration and model projection to one execution-owned memory reservation.
// Pattern: Runtime resource ownership boundary.
// Usage: Memory refresh and request assembly reserve before allocating model context representations.
// Related: internal/processmemory/context.go, internal/agent/run/memory.go, internal/agent/run/start.go.
package run

import (
	"context"

	contextbuild "github.com/keonho-kim/meca/internal/agent/context"
)

func (s *execution) contextBaseBytes() int64 {
	bytes := int64(len(s.system) + len(s.memoryContext) + len(s.checkpoint.Objective))
	for _, read := range s.skillReads {
		bytes += int64(read.ContextBytes())
	}
	return bytes
}

func (s *execution) reserveContext(bytes int64) error {
	if s.contextMemory == nil {
		return nil
	}
	// Account source values, sanitized projections and request serialization.
	// This logical bound complements input/result budgets; it is not an OS quota.
	return s.contextMemory.Resize(3 * (bytes + s.contextBaseBytes()))
}

func (s *execution) loadContext(ctx context.Context, through int64) error {
	s.messages = nil
	s.projection = contextbuild.Projection{}
	if err := s.reserveContext(0); err != nil {
		return err
	}
	messages, err := s.runner.Journal.ContextMessages(ctx, s.runner.Session, through, s.limits.MaxContextBytes, s.runner.ImageInput, func(bytes int) error { return s.reserveContext(int64(bytes)) })
	if err != nil {
		return err
	}
	s.messages = messages
	return nil
}
