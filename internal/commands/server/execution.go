// Purpose: Execute native tasks under a process-owned lifetime and confirm their terminal persistence.
// Pattern: Joined request lifecycle with one durable completion boundary.
// Usage: Start launches one request; observer disconnect never cancels engine work.
// Related: internal/commands/server/tasks.go, internal/commands/server/completion.go.
package server

import (
	"context"
	"errors"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	agentsession "github.com/keonho-kim/meca/internal/agent/session"
	"github.com/keonho-kim/meca/internal/agent/work"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/workspacegate"
)

func (s *Service) launch(c *Conversation, task Task, text string, images []media.Image, release func()) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx.Err() != nil {
		return Task{}, faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerShutDownAfterAdmissionQueryThe)
	}
	model, err := c.Model.Borrow()
	if err != nil {
		return Task{}, err
	}
	ctx, cancel := context.WithCancel(s.ctx)
	r := &request{releaseCapacity: release, teamEnabled: c.TeamEnabled, originals: make(map[string]*draftOriginal), started: time.Now(), task: task, cancel: cancel, approvals: map[string]*pendingApproval{}, steeringReady: make(chan struct{}, 1), drafts: map[string]*draftState{}, model: model, conversation: c}
	r.task.Status = "running"
	r.task.Persistence = PersistencePending
	if s.active == nil {
		s.active = make(map[string]*request)
	}
	s.active[r.task.ID] = r
	if s.bySession == nil {
		s.bySession = make(map[string]*request)
	}
	s.bySession[task.Session] = r
	options := agentrequest.Options{Runtime: c.Runtime, Generator: model.Generator(), Text: text, Images: images}
	options.Config = s.currentConfig()
	options.PollSteering = func() []string { return s.pollSteering(r) }
	options.SteeringReady = r.steeringReady
	s.publish(Event{Task: task.ID, Record: eventKind("request_started")})
	s.workers.Add(1)
	go s.execute(ctx, r, c, options)
	return r.task, nil
}

func (s *Service) execute(ctx context.Context, r *request, c *Conversation, options agentrequest.Options) {
	defer s.workers.Done()
	defer r.releaseCapacity()
	defer func() {
		if err := r.closeDrafts(); err != nil {
			s.mu.Lock()
			s.closeErr = errors.Join(s.closeErr, err)
			s.mu.Unlock()
		}
	}()
	gate := s.options.Effects
	if gate == nil {
		gate = &s.workspace
	}
	ctx, releaseWorkspace := workspacegate.Bind(ctx, gate)
	defer releaseWorkspace()
	defer r.cancel()
	defer r.model.Close()
	started := time.Now()
	ctx = diagnostics.WithRedactor(ctx, options.Vault.Redact)
	ctx = diagnostics.WithFields(ctx, "component", "server", "task_id", r.task.ID, "session_id", r.task.Session)
	diagnostics.Log(ctx, diagcatalog.RequestStarted)
	emit := func(e record.Event) error { s.emit(r, e); return nil }
	var result run.Result
	err := faults.Guard(func() error {
		if !r.teamEnabled {
			var err error
			input := agentrequest.New(options)
			options.Images = nil
			result, err = agentsession.Execute(ctx, input, s.approver(r), emit)
			return err
		}
		value, err := c.Team.Run(ctx, options, s.approver(r), emit)
		result.Status, result.Text, result.End = value.Status, value.Text, value.End
		return err
	})
	s.mu.Lock()
	if faults.IsPanic(err) {
		s.executionErr = err
	}
	task := r.task
	task.Status = string(result.Status)
	task.Outcome = string(result.End.Status)
	if task.Status == "" || task.Status == "running" {
		task.Status = diagcatalog.Interrupted
	}
	if err != nil {
		task.Error = clipped(faults.PublicMessage(err, options.Vault.Redact), 4096)
	}
	task.Code = faults.Code(err)
	task.Persistence = PersistencePending
	s.mu.Unlock()
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	saveErr := s.options.Ledger.Finish(flush, task)
	cancel()
	saveErr = s.confirmCompletion(ctx, task, saveErr)
	task.Persistence = PersistenceConfirmed
	if saveErr != nil {
		task.Persistence = PersistenceUncertain
		task.Code = diagcatalog.CompletionNotSaved
	}
	diagnostics.Log(ctx, diagcatalog.RequestFinished, "status", task.Status, "duration_ms", time.Since(started).Milliseconds(), "error_code", task.Code, "persistence", task.Persistence)
	if saveErr != nil {
		diagnostics.Log(ctx, diagcatalog.CompletionSaveFailed, "error_code", diagnostics.Code(saveErr))
	}
	s.mu.Lock()
	if saveErr != nil {
		task.Error = clipped(diagcatalog.Text(diagcatalog.CompletionUncertain), 4096)
		s.closeErr = errors.Join(s.closeErr, saveErr)
		s.completionErr = saveErr
	}
	s.last = &task
	r.steering, r.approvals, r.drafts = nil, nil, nil
	terminal := eventKind("request_finished")
	stats := r.statistics()
	terminal.Statistics = &stats
	terminal.ErrorCode = task.Code
	r.releaseCapacity()
	end := &work.End{Status: result.End.Status, Reason: clipped(options.Vault.Redact(result.End.Reason), 4096)}
	s.publish(Event{Outcome: end, Session: task.Session, Task: task.ID, Status: task.Status, Error: task.Error, Record: terminal})
	delete(s.active, task.ID)
	delete(s.bySession, task.Session)
	if len(s.active) == 0 {
		s.active, s.bySession = nil, nil
	}
	var retired []*Conversation
	if !s.changing {
		retired = s.idleJobs(c)
	}
	s.mu.Unlock()
	for _, old := range retired {
		if err := s.closeReadersAndConversation(old); err != nil {
			s.mu.Lock()
			s.closeErr = errors.Join(s.closeErr, err)
			s.mu.Unlock()
		}
	}
}
