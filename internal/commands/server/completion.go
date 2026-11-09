// Purpose: Confirm uncertain completion writes by reading their durable result without replaying work.
// Pattern: Cancellation-independent readback boundary.
// Usage: The request owner verifies a failed final write before publishing persistence state.
// Related: internal/commands/server/execution.go, internal/storage/serverrequests/ledger.go.
package server

import (
	"context"
	"errors"
	"time"
)

func (s *Service) confirmCompletion(ctx context.Context, task Task, cause error) error {
	if cause == nil {
		return nil
	}
	verify, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	actual, err := s.options.Ledger.Get(verify, task.ID)
	if err == nil && actual.ID == task.ID && actual.Session == task.Session && actual.Status == task.Status && actual.Error == task.Error && actual.Code == task.Code && actual.Persistence == PersistenceConfirmed {
		return nil
	}
	return errors.Join(cause, err)
}
