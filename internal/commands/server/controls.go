// Purpose: Bind user decisions and steering to one live request without replaying stale approvals.
// Pattern: Request-owned control boundary.
// Usage: Runtime approval callbacks wait here; authenticated commands resolve them once.
// Related: internal/commands/server/tasks.go, internal/agent/run/prepare_execution.go.
package server

import (
	"context"
	"crypto/rand"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/run"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

type pendingApproval struct {
	value    tool.Approval
	ctx      context.Context
	decision chan bool
}

func (s *Service) approver(r *request) run.Approver {
	return func(ctx context.Context, value tool.Approval) (bool, error) {
		id := "approval_" + rand.Text()
		pending := &pendingApproval{value: value, ctx: ctx, decision: make(chan bool, 1)}
		s.mu.Lock()
		if ctx.Err() != nil {
			s.mu.Unlock()
			return false, ctx.Err()
		}
		r.approvals[id] = pending
		s.publish(Event{Task: r.task.ID, ApprovalID: id, Record: record.Event{Kind: "approval_required", Approval: &value, CallID: value.ID}})
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			delete(r.approvals, id)
			s.publish(Event{Task: r.task.ID, ApprovalID: id, Record: eventKind("approval_closed")})
			s.mu.Unlock()
		}()
		select {
		case decision := <-pending.decision:
			if err := ctx.Err(); err != nil {
				return false, err
			}
			return decision, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
}

func (s *Service) Approve(task, id string, allow bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.currentRequest(task)
	if err != nil {
		return err
	}
	pending := r.approvals[id]
	if pending == nil || pending.ctx.Err() != nil || r.task.Status != "running" {
		return failure("conflict", "approval is stale or already answered")
	}
	delete(r.approvals, id)
	pending.decision <- allow
	return nil
}

func (s *Service) Steer(task, text string) error {
	if strings.TrimSpace(text) == "" || !utf8.ValidString(text) {
		return failure("invalid_argument", "a UTF-8 instruction is required")
	}
	if err := checkText(text, 1<<20); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.currentRequest(task)
	if err != nil {
		return err
	}
	if r.task.Status != "running" {
		return failure("conflict", "request is canceling")
	}
	if len(r.steering) >= 16 || r.steeringBytes+len(text) > 1<<20 {
		return failure("resource_exhausted", "steering queue is full")
	}
	r.steering = append(r.steering, text)
	r.steeringBytes += len(text)
	select {
	case r.steeringReady <- struct{}{}:
	default:
	}
	return nil
}

func (s *Service) pollSteering(r *request) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	values := r.steering
	r.steering, r.steeringBytes = nil, 0
	return values
}
