// Purpose: Query compatible response state while preserving retention and uncertain completion semantics.
// Pattern: Application state read boundary.
// Usage: Compatible lookup and awaiting use the same durable response state projection.
// Related: internal/commands/server/response.go, internal/commands/server/response_wait.go.
package server

import (
	"context"

	"github.com/keonho-kim/meca/internal/agent/work"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (r Response) task() Task {
	status := r.TaskStatus
	if status == "" {
		status = diagcatalog.Interrupted
	}
	persistence := PersistenceConfirmed
	if r.Status == "queued" || r.Status == "in_progress" {
		persistence = PersistencePending
		if r.TaskStatus == "" {
			status = "accepted"
		}
	}
	return Task{ID: r.Task, Session: r.Session, Status: status, Error: r.Error, Code: r.Code, Persistence: persistence}
}

// ResponseState hides nonretained results only from public lookup; POST callers
// can still await their accepted task and an idempotent retry can recover it.
func (s *Service) ResponseState(ctx context.Context, id string, lookup bool) (Response, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	s.mu.Lock()
	if s.closed || s.ctx.Err() != nil {
		s.mu.Unlock()
		return Response{}, faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerIsShuttingDown)
	}
	if r := s.active; r != nil && r.response != nil && r.response.ID == id {
		value := *r.response
		s.mu.Unlock()
		if lookup && (!value.Stored || value.Kind != "response") {
			return Response{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerResponseWasNotFound)
		}
		return value, nil
	}
	if s.last != nil && len(s.last.ID) >= 5 && s.last.Code == diagcatalog.CompletionNotSaved && id == "resp_"+s.last.ID[5:] {
		s.mu.Unlock()
		return Response{}, faults.New(diagcatalog.CompletionNotSaved, diagcatalog.CompletionUncertain)
	}
	s.mu.Unlock()
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return Response{}, err
	}
	defer done()
	if s.options.Responses == nil {
		return Response{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerCompatibleResultsAreUnavailable)
	}
	value, err := s.options.Responses.ReadResponse(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if value.Scope != s.scope || lookup && (!value.Stored || value.Kind != "response") {
		return Response{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerResponseWasNotFound)
	}
	s.mu.Lock()
	live := s.active != nil && s.active.response != nil && s.active.response.ID == id || s.submitting == id
	s.mu.Unlock()
	if !live && (value.Status == "queued" || value.Status == "in_progress") {
		// Completion is persisted before active is cleared. Re-read after
		// observing that transition rather than classifying an older file view.
		value, err = s.options.Responses.ReadResponse(ctx, id)
		if err != nil {
			return Response{}, err
		}
	}
	if !live && (value.Status == "queued" || value.Status == "in_progress") {
		value.Status, value.Error = "failed", diagcatalog.Text(diagcatalog.ResponseInterrupted)
		value.TaskStatus = diagcatalog.Interrupted
		value.Code = diagcatalog.Interrupted
		value.End = work.End{Status: work.Interrupted, Reason: value.Error}
	}
	return value, nil
}
