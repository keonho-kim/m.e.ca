// Purpose: Reserve mutations against an explicit session without changing another client's selection.
// Pattern: Session-scoped state transition.
// Usage: Model, team, reference and close commands use beginTargetChange.
// Related: internal/commands/server/operations.go, internal/commands/server/lifecycle.go.
package server

import (
	"context"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (s *Service) beginTargetChange(ctx context.Context, id string) (sessionChange, error) {
	if !sessionstate.ValidID(id) {
		return sessionChange{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerExplicitSession)
	}
	change, err := s.beginChange(ctx)
	if err != nil {
		return change, err
	}
	s.mu.Lock()
	c := s.ownedConversation(id)
	s.mu.Unlock()
	if c == nil {
		c, err = s.options.Open(change.ctx, id, false, change.config)
		if err != nil {
			change.finish()
			return sessionChange{}, err
		}
		s.mu.Lock()
		s.jobs[id] = c
		s.mu.Unlock()
	}
	change.conversation = c
	return change, nil
}

func (s *Service) targetState(c *Conversation) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionSnapshot(c)
}

// sessionSnapshot requires s.mu and never includes another session's work.
func (s *Service) sessionSnapshot(c *Conversation) Snapshot {
	value := s.baseSnapshot()
	if r := s.sessionRequest(c.Runtime.Session.ID); r != nil {
		value = s.snapshotPage(r.task.ID, "", 1)
	} else if s.last != nil && s.last.Session == c.Runtime.Session.ID {
		copy := *s.last
		value.Task = &copy
	}
	value.Session, value.Selection, value.Team = c.Runtime.Session, c.Runtime.Selection, c.TeamEnabled
	value.TotalTasks = len(value.Tasks)
	value.NextTask = ""
	return value
}

func (s *Service) baseSnapshot() Snapshot {
	permission, _ := s.options.Config.Permission.MarshalText()
	return Snapshot{Language: string(s.options.Config.Language), Cursor: s.feed.cursor(), Permission: string(permission), Concurrency: s.concurrency()}
}
