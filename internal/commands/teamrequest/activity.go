// Purpose: Own the narrow registry of live command sources and expose bounded metadata snapshots.
// Pattern: Synchronized resource projection.
// Usage: The terminal clock polls live command metadata; closed peers remove their executor.
// Related: internal/commands/teamrequest/request.go, internal/tui/session/team_events.go.
package teamrequest

import toolapi "github.com/keonho-kim/meca/internal/agent/tool"

// CommandActivity exposes synchronized metadata and remains safe during or after Close.
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
	type activeSource struct {
		id     string
		source commandSource
	}
	sources := make([]activeSource, 0, len(s.active))
	for id, source := range s.active {
		sources = append(sources, activeSource{id, source})
	}
	s.mu.Unlock()
	var result map[string][]toolapi.CommandRun
	for _, current := range sources {
		if commands := current.source.CommandActivity(); len(commands) > 0 {
			if result == nil {
				result = map[string][]toolapi.CommandRun{}
			}
			result[current.id] = commands
		}
	}
	return result
}
