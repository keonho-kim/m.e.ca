// Purpose: Retire delivered evidence and expose the tools' current unread cursors.
// Pattern: Execution-owned acknowledgement at a completed model decision.
// Usage: A completed decision releases its observations; each request gets the live cursor state.
// Related: internal/agent/run/loop.go, internal/agent/run/context_memory.go, internal/tools/browser/paging.go.
package run

import (
	"context"
	"encoding/json"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) acknowledgeObservations(through int) {
	for i := s.observationReadThrough; i < through; i++ {
		m := &s.messages[i]
		if !m.TransientObservation {
			continue
		}
		m.Content = toolapi.ObservationRead
		m.Images = nil
		m.TransientObservation = false
		s.projection.ForgetMessage(i)
	}
	s.observationReadThrough = through
}

func (s *execution) restoreObservationReads() {
	s.observationReadThrough = 0
	for i := len(s.messages) - 1; i >= 0; i-- {
		m := s.messages[i]
		if m.Role == "assistant" && !m.Incomplete {
			s.acknowledgeObservations(i)
			return
		}
	}
}

func (s *execution) observationContinuations(ctx context.Context) (string, error) {
	source, ok := s.runner.Tools.(toolapi.ObservationCursorSource)
	if !ok {
		return "", nil
	}
	cursors, err := source.ObservationCursors(ctx)
	if err != nil || len(cursors) == 0 {
		return "", err
	}
	body, err := json.Marshal(cursors)
	if err != nil {
		return "", err
	}
	return "Unread observations. Continue only if needed by calling the listed tool with result_id, revision and next_offset as offset:\n" + string(body), nil
}
