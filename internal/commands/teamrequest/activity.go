// Purpose: Own the narrow registry of live command sources and expose bounded metadata snapshots.
// Pattern: Synchronized resource projection.
// Usage: The terminal clock polls live command metadata; closed peers remove their executor.
// Related: internal/commands/teamrequest/request.go, internal/tui/session/team_events.go.
package teamrequest

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type commandSource interface{ CommandActivity() []toolapi.CommandRun }

func (s *Service) track(id string, executor commandSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if executor == nil {
		delete(s.active, id)
		if len(s.active) == 0 {
			s.active = nil
		}
		return
	}
	if s.active == nil {
		s.active = map[string]commandSource{}
	}
	s.active[id] = executor
}
func (s *Service) CommandActivity() map[string][]toolapi.CommandRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result map[string][]toolapi.CommandRun
	for id, executor := range s.active {
		if commands := executor.CommandActivity(); len(commands) > 0 {
			if result == nil {
				result = map[string][]toolapi.CommandRun{}
			}
			result[id] = commands
		}
	}
	return result
}
