// Purpose: Admit durable requests and own their lifetime independently of client connections.
// Pattern: Explicit request state transitions.
// Usage: Start, TaskStatus and Cancel are shared by every server transport.
// Related: internal/commands/server/controls.go, internal/storage/serverrequests/ledger.go.
package server

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	conversationruntime "github.com/keonho-kim/meca/internal/commands/conversation"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type request struct {
	model         *conversationruntime.Model
	conversation  *Conversation
	task          Task
	cancel        context.CancelFunc
	approvals     map[string]*pendingApproval
	steering      []string
	steeringBytes int
	steeringReady chan struct{}
	drafts        map[string]*draftState
	response      *Response
	done          chan struct{}
}

func (s *Service) Start(ctx context.Context, session, key, text string) (Task, error) {
	if strings.TrimSpace(text) == "" || !utf8.ValidString(text) {
		return Task{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerAUTFPromptIsRequired)
	}
	if err := checkText(text, 16<<20); err != nil {
		return Task{}, err
	}
	if len(key) < 8 || len(key) > 128 || !utf8.ValidString(key) || strings.ContainsAny(key, "\r\n\x00") {
		return Task{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerRequestKeyMustContainToUTFBytes)
	}
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return Task{}, err
	}
	defer done()
	s.mu.Lock()
	err = s.requireConversation()
	if err == nil && session != s.conversation.Runtime.Session.ID {
		err = faults.New(diagcatalog.Conflict, diagcatalog.ServerRequestSessionIsNotTheSelectedSession)
	}
	s.mu.Unlock()
	if err != nil {
		return Task{}, err
	}
	// Ledger I/O never holds the runtime publication or cancellation mutex.
	if task, found, err := s.existingRequest(ctx, session, key, text); err != nil || found {
		return task, err
	}
	change, err := s.beginChange(ctx)
	if err != nil {
		var busy *faults.Error
		if errors.As(err, &busy) && busy.ErrorCode() == diagcatalog.Conflict {
			if task, found, cause := s.existingRequest(ctx, session, key, text); cause != nil || found {
				return task, cause
			}
		}
		return Task{}, err
	}
	defer change.finish()
	c := change.conversation
	if c == nil || c.Runtime.Session.ID != session {
		return Task{}, faults.New(diagcatalog.Conflict, diagcatalog.ServerSelectedSessionChangedBeforeAdmission)
	}
	s.mu.Lock()
	s.admitting = TaskID(session, key)
	s.mu.Unlock()
	if c.Model == nil {
		selected := c.Runtime.Selection
		if selected.Model == "" {
			return Task{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerSelectAModelBeforeStartingARequest)
		}
		if err := s.selectModel(change, ModelSettings{Provider: string(selected.Connection.Kind), Endpoint: selected.Connection.Endpoint, Name: selected.Model, Thinking: selected.Thinking, Effort: selected.ReasoningEffort}); err != nil {
			return Task{}, err
		}
	}
	task, duplicate, err := s.options.Ledger.Admit(change.ctx, session, key, text)
	if err != nil || duplicate {
		return task, err
	}
	return s.launch(c, task, text, nil, nil)
}

func (s *Service) TaskStatus(ctx context.Context, id string) (Task, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return Task{}, err
	}
	defer done()
	s.mu.Lock()
	if s.active != nil && s.active.task.ID == id {
		value := s.active.task
		s.mu.Unlock()
		return value, nil
	}
	if s.last != nil && s.last.ID == id {
		value := *s.last
		s.mu.Unlock()
		return value, nil
	}
	s.mu.Unlock()
	value, err := s.options.Ledger.Get(ctx, id)
	if errors.Is(err, ErrTaskNotFound) && s.options.Responses != nil && len(id) == 69 && strings.HasPrefix(id, "task_") {
		response, cause := s.ResponseState(ctx, "resp_"+id[5:], false)
		if cause == nil && response.Task == id {
			return response.task(), nil
		}
		var public *faults.Error
		if cause != nil && (!errors.As(cause, &public) || public.ErrorCode() != diagcatalog.NotFound) {
			return Task{}, cause
		}
	}
	if err != nil {
		return Task{}, err
	}
	return s.observedTask(value), nil
}

func (s *Service) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.currentRequest(id)
	if err != nil {
		return err
	}
	r.task.Status = "canceling"
	r.cancel()
	s.publish(Event{Task: id, Record: eventKind("cancel_requested")})
	return nil
}

func (s *Service) currentRequest(id string) (*request, error) {
	if s.closed {
		return nil, faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerIsShuttingDown)
	}
	if s.active == nil || s.active.task.ID != id {
		return nil, faults.New(diagcatalog.Conflict, diagcatalog.ServerRequestIsNoLongerActive)
	}
	return s.active, nil
}
