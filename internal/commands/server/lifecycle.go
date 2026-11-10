// Purpose: Transfer selected session resources and publish every irreversible state change before cleanup.
// Pattern: Explicit session lifecycle transitions.
// Usage: Select, CloseSession and Close detach resources under the mutex and close them outside it.
// Related: internal/commands/server/operations.go, internal/commands/server/events.go.
package server

import (
	"context"
	"errors"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (s *Service) Sessions(ctx context.Context, cursor string, limit int) ([]sessionstate.Conversation, string, error) {
	if limit < 1 || limit > 100 || len(cursor) > 1024 {
		return nil, "", faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerSessionPageRequiresLimitToAndA)
	}
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return nil, "", err
	}
	defer done()
	values, next, err := s.options.List(ctx, cursor, limit)
	if errors.Is(err, sessionstate.ErrCursor) {
		return nil, "", faults.New(diagcatalog.InvalidArgument, diagcatalog.WrappedCause, err)
	}
	if errors.Is(err, sessionstate.ErrPageLimit) {
		return nil, "", faults.New(diagcatalog.ResourceExhausted, diagcatalog.WrappedCause, err)
	}
	return values, next, err
}

func (s *Service) Select(ctx context.Context, id string, create bool) (Snapshot, error) {
	if create == (id != "") || !create && !sessionstate.ValidID(id) {
		return Snapshot{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerChooseCreateOrAValidExistingSession)
	}
	change, err := s.beginChange(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	defer change.finish()
	if !create && change.conversation != nil && change.conversation.Runtime.Session.ID == id {
		return s.State(), nil
	}
	s.mu.Lock()
	existing := s.jobs[id]
	if !create && existing != nil && existing.Runtime.Session.ID == id {
	} else {
		existing = nil
	}
	s.mu.Unlock()
	if existing != nil {
		return s.replaceConversation(change, existing)
	}
	next, err := s.options.Open(change.ctx, id, create, change.config)
	if err != nil {
		if errors.Is(err, sessionstate.ErrNotFound) {
			return Snapshot{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerSessionWasNotFoundInThisProject)
		}
		return Snapshot{}, err
	}
	if err := change.ctx.Err(); err != nil {
		return Snapshot{}, errors.Join(err, closeConversation(next))
	}
	return s.replaceConversation(change, next)
}

func (s *Service) replaceConversation(change sessionChange, next *Conversation) (Snapshot, error) {
	s.mu.Lock()
	owned := s.ownedConversation(next.Runtime.Session.ID) == next
	s.mu.Unlock()
	discard := func() error {
		if owned {
			return nil
		}
		return closeConversation(next)
	}
	if change.conversation != nil {
		s.mu.Lock()
		busy := s.sessionRequest(change.conversation.Runtime.Session.ID) != nil
		if busy {
			if s.jobs == nil {
				s.jobs = make(map[string]*Conversation)
			}
			s.jobs[change.conversation.Runtime.Session.ID] = change.conversation
		}
		s.detachConversation()
		s.mu.Unlock()
		if !busy {
			if err := s.closeReadersAndConversation(change.conversation); err != nil {
				return Snapshot{}, errors.Join(err, discard())
			}
		}
	}
	s.mu.Lock()
	if s.closed || change.ctx.Err() != nil {
		s.mu.Unlock()
		return Snapshot{}, errors.Join(context.Canceled, discard())
	}
	delete(s.jobs, next.Runtime.Session.ID)
	s.conversation = next
	s.publish(Event{Record: eventKind("session_opened")})
	state := s.snapshot()
	s.mu.Unlock()
	return state, nil
}

func (s *Service) CloseSession(ctx context.Context, id string) error {
	if !sessionstate.ValidID(id) {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerExplicitSession)
	}
	change, err := s.beginChange(ctx)
	if err != nil {
		return err
	}
	defer change.finish()
	s.mu.Lock()
	change.conversation = s.ownedConversation(id)
	s.mu.Unlock()
	if change.conversation == nil {
		return nil
	}
	if err := s.sessionIdle(change.conversation); err != nil {
		return err
	}
	s.mu.Lock()
	if s.conversation == change.conversation {
		s.detachConversation()
	}
	delete(s.jobs, id)
	s.mu.Unlock()
	return s.closeReadersAndConversation(change.conversation)
}

// detachConversation requires s.mu. Readers are canceled outside the mutex.
func (s *Service) detachConversation() {
	old := s.conversation
	s.conversation, s.last = nil, nil
	id := ""
	if old != nil {
		id = old.Runtime.Session.ID
	}
	s.publish(Event{Session: id, Record: eventKind("session_closed")})
}

func (s *Service) closeReadersAndConversation(c *Conversation) error {
	if c != nil && c.stopReaders != nil {
		c.stopReaders()
	}
	if c != nil {
		c.readers.Wait()
	}
	return closeConversation(c)
}

func closeConversation(c *Conversation) error {
	if c == nil {
		return nil
	}
	c.Model.Close()
	return c.Close()
}

func (s *Service) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		<-s.closeDone
		return s.closeErr
	}
	s.closed = true
	s.cancel()
	s.closeAttachments()
	s.mu.Unlock()
	s.workers.Wait()
	s.mu.Lock()
	c := s.conversation
	var jobs []*Conversation
	for _, job := range s.jobs {
		if job != c {
			jobs = append(jobs, job)
		}
	}
	s.jobs = nil
	s.detachConversation()
	s.mu.Unlock()
	err := s.closeReadersAndConversation(c)
	for _, job := range jobs {
		err = errors.Join(err, s.closeReadersAndConversation(job))
	}
	s.mu.Lock()
	s.closeErr = errors.Join(s.closeErr, err)
	s.feed.wake()
	close(s.closeDone)
	s.mu.Unlock()
	return s.closeErr
}
