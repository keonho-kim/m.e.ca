// Purpose: Execute admitted native and compatible tasks under the same process-owned lifetime.
// Pattern: Shared request lifecycle and terminal result transition.
// Usage: Start and Submit call launch only after durable admission and session reservation.
// Related: internal/commands/server/tasks.go, internal/commands/server/response.go.
package server

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	agentsession "github.com/keonho-kim/meca/internal/agent/session"
	"github.com/keonho-kim/meca/internal/agent/work"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
)

func (s *Service) launch(c *Conversation, task Task, text string, response *Response, clientContext []model.Message) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx.Err() != nil {
		return Task{}, failure("unavailable", "server shut down after admission; query the task before retrying")
	}
	model, err := c.Model.Borrow()
	if err != nil {
		return Task{}, err
	}
	ctx, cancel := context.WithCancel(s.ctx)
	r := &request{task: task, cancel: cancel, approvals: map[string]*pendingApproval{}, steeringReady: make(chan struct{}, 1), drafts: map[string]*draftState{}, response: response, done: make(chan struct{}), model: model, conversation: c}
	r.task.Status = "running"
	s.active = r
	options := agentrequest.Options{Runtime: c.Runtime}
	options.Generator = model.Generator()
	options.Config, options.Text = s.options.Config, text
	if c.TeamEnabled {
		options.ClientContext = clientContext
	}
	options.PollSteering = func() []string { return s.pollSteering(r) }
	options.SteeringReady = r.steeringReady
	s.publish(Event{Task: task.ID, Record: eventKind("request_started")})
	s.workers.Add(1)
	go s.execute(ctx, r, c, options)
	return r.task, nil
}

func (s *Service) execute(ctx context.Context, r *request, c *Conversation, options agentrequest.Options) {
	defer s.workers.Done()
	defer r.cancel()
	defer r.model.Close()
	var totals record.UsageTotals
	var usageMu sync.Mutex
	missingUsage := false
	emit := func(e record.Event) error {
		if r.response != nil && (e.Kind == "usage" || e.Kind == "generation_finished") {
			usageMu.Lock()
			if e.Kind == "usage" {
				totals.Add(e)
			} else {
				missingUsage = true
			}
			usageMu.Unlock()
		}
		s.emit(r, e)
		return nil
	}
	approve := s.approver(r)
	if r.response != nil {
		approve = agentrequest.RejectApproval
	}
	var result run.Result
	var err error
	if !c.TeamEnabled {
		result, err = agentsession.Execute(ctx, agentrequest.New(options), approve, emit)
	} else {
		value, cause := c.Team.Run(ctx, options, approve, emit)
		result.Status, result.Text, result.End, err = value.Status, value.Text, value.End, cause
	}
	s.mu.Lock()
	task := r.task
	task.Status = string(result.Status)
	if task.Status == "" || task.Status == "running" {
		task.Status = "interrupted"
	}
	if err != nil {
		task.Error = clipped(options.Vault.Redact(err.Error()), 4096)
	}
	if result.End.Status == "" {
		result.End = work.End{Status: work.Failed, Reason: task.Error}
	}
	s.mu.Unlock()
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	var saveErr error
	if r.response != nil {
		value := *r.response
		value.TaskStatus = task.Status
		value.End, value.Error, value.Completed = result.End, task.Error, max(value.Created, time.Now().Unix())
		value.End.Reason = clipped(options.Vault.Redact(value.End.Reason), 4096)
		value.Status = "failed"
		value.Code = "execution_failed"
		if errors.Is(err, agentrequest.ErrApprovalRequired) {
			value.Code = "approval_required"
		}
		switch result.End.Status {
		case work.Completed:
			if err == nil {
				value.Status, value.Code = "completed", ""
			}
		case work.Exhausted:
			value.Status = "incomplete"
		case work.Interrupted:
			value.Status = "cancelled"
		}
		if totals.Received && !missingUsage {
			u := totals.Usage.Clone()
			value.Usage = &u
		}
		value.Through, saveErr = c.Output.OutputPosition(flush, task.Session)
		if saveErr == nil {
			saveErr = s.options.Responses.SaveResponse(flush, value)
		}
	} else {
		saveErr = s.options.Ledger.Finish(flush, task)
	}
	cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if saveErr != nil {
		task.Error = clipped(task.Error+"; could not persist request completion", 4096)
		s.closeErr = errors.Join(s.closeErr, saveErr)
	}
	s.last, s.active = &task, nil
	close(r.done)
	r.steering, r.approvals, r.drafts = nil, nil, nil
	s.publish(Event{Session: task.Session, Task: task.ID, Status: task.Status, Error: task.Error, Record: eventKind("request_finished")})
}
