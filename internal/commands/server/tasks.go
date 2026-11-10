// Purpose: Admit durable requests and own their lifetime independently of client connections.
// Pattern: Explicit request state transitions.
// Usage: Start, TaskStatus and Cancel are shared by every server transport.
// Related: internal/commands/server/controls.go, internal/storage/serverrequests/ledger.go.
package server

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	conversationruntime "github.com/keonho-kim/meca/internal/commands/conversation"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type request struct {
	teamEnabled      bool
	releaseCapacity  func()
	draftCleanup     error
	draftGeneration  uint64
	started          time.Time
	activity         map[string]Event
	completed        modelapi.TokenUsage
	liveUsage        map[string]usageSample
	generationMillis int64
	outputMu         sync.Mutex
	originals        map[string]*draftOriginal
	sequence         uint64
	model            *conversationruntime.Model
	conversation     *Conversation
	task             Task
	cancel           context.CancelFunc
	approvals        map[string]*pendingApproval
	steering         []string
	steeringBytes    int
	steeringReady    chan struct{}
	drafts           map[string]*draftState
}

func (s *Service) Start(ctx context.Context, session, key, text string, images ...media.Image) (Task, error) {
	return s.start(ctx, session, key, text, InputDigest(text, images), images...)
}

func (s *Service) start(ctx context.Context, session, key, text string, identity [32]byte, images ...media.Image) (Task, error) {
	if !sessionstate.ValidID(session) {
		return Task{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerRequestSessionIsNotTheSelectedSession)
	}
	if strings.TrimSpace(text) == "" && len(images) == 0 || !utf8.ValidString(text) {
		return Task{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerAUTFPromptIsRequired)
	}
	if err := checkText(text, MaxPromptBytes); text != "" && err != nil {
		return Task{}, err
	}
	if err := media.CheckInput(images); err != nil {
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
	// Ledger I/O never holds the runtime publication or cancellation mutex.
	if task, found, err := s.existingRequest(ctx, session, key, identity); err != nil || found {
		return task, err
	}
	change, err := s.beginAdmission(ctx)
	if err != nil {
		var busy *faults.Error
		if errors.As(err, &busy) && (busy.ErrorCode() == diagcatalog.Conflict || busy.ErrorCode() == diagcatalog.ResourceExhausted) {
			if task, found, cause := s.existingRequest(ctx, session, key, identity); cause != nil || found {
				return task, cause
			}
		}
		return Task{}, err
	}
	defer change.finish()
	s.mu.Lock()
	c := s.ownedConversation(session)
	s.mu.Unlock()
	change.conversation = c
	if c == nil {
		c, err = s.options.Open(change.ctx, session, false, change.config)
		if err != nil {
			return Task{}, err
		}
		change.conversation = c
		s.mu.Lock()
		s.jobs[session] = c
		s.mu.Unlock()
	}
	if err := s.sessionIdle(c); err != nil {
		return Task{}, err
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
	if err := checkImageModel(c, images); err != nil {
		return Task{}, err
	}
	release, available := s.options.Capacity.reserve()
	if !available {
		return Task{}, faults.New(diagcatalog.ResourceExhausted, diagcatalog.ServerTaskCapacity)
	}
	transferred := false
	defer func() {
		if !transferred {
			release()
		}
	}()
	task, duplicate, err := s.options.Ledger.Admit(change.ctx, session, key, identity)
	if err != nil || duplicate {
		return task, err
	}
	value, err := s.launch(c, task, text, images, release)
	transferred = err == nil
	return value, err
}

func (s *Service) TaskStatus(ctx context.Context, id string) (Task, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return Task{}, err
	}
	defer done()
	s.mu.Lock()
	if r := s.active[id]; r != nil {
		value := r.task
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
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return Task{}, faults.New(diagcatalog.NotFound, diagcatalog.WrappedCause, err)
		}
		return Task{}, err
	}
	value = s.observedTask(value)
	// Reuse bounded terminal facts already retained by the process replay log.
	// The durable ledger deliberately stores admission, not a second run result.
	if value.Outcome == "" {
		value.Outcome = s.feed.outcome(s.options.WorkspaceID, id)
	}
	return value, nil
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
	r := s.active[id]
	if r == nil {
		return nil, faults.New(diagcatalog.Conflict, diagcatalog.ServerRequestIsNoLongerActive)
	}
	return r, nil
}
