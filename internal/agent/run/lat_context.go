// Purpose: Complete independent review evidence without waiting on an unnecessary summary.
// Pattern: Safe-boundary preparation with cancellable single-job catch-up.
// Usage: Final LAT review calls this after commands settle and before its model request.
// Related: internal/agent/run/lat_memory.go, internal/agent/lat/memory/tail.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/keonho-kim/meca/internal/agent/lat"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
)

func (s *execution) prepareLATContext(ctx context.Context, input lat.Input) ([]byte, error) {
	budget, err := lat.SourceBudget(input)
	if err != nil {
		return nil, err
	}
	if s.latMemory == nil {
		return nil, fmt.Errorf("independent review memory is unavailable")
	}
	for {
		if err := s.reviewInstructions(); err != nil {
			return nil, errors.Join(err, s.joinLATMemory(ctx))
		}
		if err := s.pollLATMemory(ctx); err != nil {
			return nil, err
		}
		body, err := s.latMemory.controller.Tail(ctx, budget, s.checkpoint.Objective)
		if err == nil {
			// A complete original delta already fits. Release model resources
			// before final review instead of awaiting optional compaction.
			if joinErr := s.joinLATMemory(ctx); joinErr != nil {
				return nil, joinErr
			}
			return body, nil
		}
		if !errors.Is(err, latmemory.ErrNeedsCompaction) {
			return nil, err
		}
		if s.latMemory.failure != nil {
			return nil, errors.Join(err, s.latMemory.failure)
		}
		if !s.activeLATMemory() {
			if startErr := s.maintainLATMemory(ctx, true); startErr != nil {
				return nil, startErr
			}
			if !s.activeLATMemory() {
				return nil, errors.Join(err, s.latMemory.failure)
			}
		}
		if err := s.waitLATMemory(ctx); err != nil {
			return nil, err
		}
	}
}

func (s *execution) waitLATMemory(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for s.activeLATMemory() {
		select {
		case <-s.latMemory.controller.Done():
			return s.pollLATMemory(ctx)
		case <-ticker.C:
			if err := s.pollLATMemory(ctx); err != nil {
				return err
			}
			if err := s.reviewInstructions(); err != nil {
				return errors.Join(err, s.joinLATMemory(ctx))
			}
		case <-ctx.Done():
			return errors.Join(ctx.Err(), s.joinLATMemory(ctx))
		}
	}
	return nil
}
