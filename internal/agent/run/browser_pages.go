// Purpose: Retire delivered browser evidence and expose the browser's current unread cursors.
// Pattern: Execution-owned acknowledgement at a completed model decision.
// Usage: A completed decision releases its observations; each request gets the live cursor state.
// Related: internal/agent/run/loop.go, internal/agent/run/context_memory.go, internal/tools/browser/paging.go.
package run

import (
	"context"
	"encoding/json"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) acknowledgeBrowserObservations(through int) {
	for i := s.browserReadThrough; i < through; i++ {
		m := &s.messages[i]
		if !m.TransientObservation {
			continue
		}
		m.Content = toolapi.BrowserObservationRead
		m.Images = nil
		m.TransientObservation = false
		s.projection.ForgetMessage(i)
	}
	s.browserReadThrough = through
}

func (s *execution) restoreBrowserReads() {
	s.browserReadThrough = 0
	for i := len(s.messages) - 1; i >= 0; i-- {
		m := s.messages[i]
		if m.Role == "assistant" && !m.Incomplete {
			s.acknowledgeBrowserObservations(i)
			return
		}
	}
}

func (s *execution) browserContinuations(ctx context.Context) (string, error) {
	source, ok := s.runner.Tools.(toolapi.BrowserCursorSource)
	if !ok {
		return "", nil
	}
	cursors, err := source.BrowserCursors(ctx)
	if err != nil || len(cursors) == 0 {
		return "", err
	}
	body, err := json.Marshal(cursors)
	if err != nil {
		return "", err
	}
	return "Unread browser observations. Continue only if needed using result_id, revision and next_offset as offset:\n" + string(body), nil
}
