// Purpose: Admit bounded independent tasks and locate their session-owned runtime state.
// Pattern: Event-owner registry with serialized preparation and explicit capacity checks.
// Usage: Start reserves admission; controls and snapshots look up active task identities.
// Related: internal/commands/server/tasks.go, internal/commands/server/operations.go, internal/commands/server/execution.go.
package server

import (
	"context"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Service) concurrency() int { return max(1, s.options.Concurrency) }

// ownedConversation requires s.mu; presentation selection does not own a task's session.
func (s *Service) ownedConversation(id string) *Conversation {
	if c := s.conversation; c != nil && c.Runtime.Session.ID == id {
		return c
	}
	return s.jobs[id]
}

// sessionRequest requires s.mu. One session keeps exactly one journal writer.
func (s *Service) sessionRequest(id string) *request {
	if len(s.bySession) != len(s.active) {
		s.bySession = make(map[string]*request, len(s.active))
		for _, r := range s.active {
			s.bySession[r.task.Session] = r
		}
	}
	return s.bySession[id]
}

// displayRequest requires s.mu; selecting another conversation never changes task ownership.
func (s *Service) displayRequest() *request {
	if s.conversation != nil {
		if r := s.sessionRequest(s.conversation.Runtime.Session.ID); r != nil {
			return r
		}
	}
	var selected *request
	for _, r := range s.active {
		if selected == nil || r.task.ID < selected.task.ID {
			selected = r
		}
	}
	return selected
}

func (s *Service) sessionIdle(c *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c != nil && s.sessionRequest(c.Runtime.Session.ID) != nil {
		return faults.New(diagcatalog.Conflict, diagcatalog.ServerSessionTaskActive)
	}
	return nil
}

func (s *Service) beginAdmission(ctx context.Context) (sessionChange, error) {
	s.mu.Lock()
	if s.admissions == nil {
		s.admissions = make(chan struct{}, 1)
	}
	gate := s.admissions
	s.mu.Unlock()
	select {
	case gate <- struct{}{}:
	case <-ctx.Done():
		return sessionChange{}, ctx.Err()
	case <-s.ctx.Done():
		return sessionChange{}, s.ctx.Err()
	}
	s.mu.Lock()
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		<-gate
		return sessionChange{}, err
	}
	if err := s.requireAvailable(); err != nil {
		s.mu.Unlock()
		<-gate
		return sessionChange{}, err
	}
	if len(s.active) >= s.concurrency() {
		s.mu.Unlock()
		<-gate
		return sessionChange{}, faults.New(diagcatalog.ResourceExhausted, diagcatalog.ServerTaskCapacity)
	}
	s.changing = true
	tracked, done := s.trackLocked(ctx)
	change := sessionChange{ctx: tracked, conversation: s.conversation, config: s.currentConfig()}
	change.finish = func() {
		s.endChange()
		<-gate
		done()
	}
	s.mu.Unlock()
	return change, nil
}
