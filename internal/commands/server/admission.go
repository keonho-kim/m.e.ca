// Purpose: Resolve durable duplicate requests across concurrent admission and completion transitions.
// Pattern: Ledger lookup followed by a short state reconciliation.
// Usage: Start retries lookups after a competing admission; TaskStatus reconciles disk observations with live ownership.
// Related: internal/commands/server/tasks.go, internal/storage/serverrequests/ledger.go.
package server

import (
	"context"
	"errors"
)

func (s *Service) existingRequest(ctx context.Context, session, key, text string) (Task, bool, error) {
	_, err := s.options.Ledger.Get(ctx, TaskID(session, key))
	if errors.Is(err, ErrTaskNotFound) {
		return Task{}, false, nil
	}
	if err != nil {
		return Task{}, false, err
	}
	// Admit checks the original digest; it must never be called for an absent
	// record while a different request or configuration owns the idle reservation.
	original, _, err := s.options.Ledger.Admit(ctx, session, key, text)
	if err != nil {
		return Task{}, true, err
	}
	return s.observedTask(original), true, nil
}

func (s *Service) observedTask(value Task) Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil && s.active.task.ID == value.ID {
		return s.active.task
	}
	if s.last != nil && s.last.ID == value.ID {
		return *s.last
	}
	if s.admitting == value.ID && value.Status == "interrupted" {
		// This process still owns the handoff after durable admission. Only an
		// unowned accepted record is evidence of an interrupted previous process.
		value.Status = "accepted"
	}
	return value
}
