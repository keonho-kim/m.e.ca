// Purpose: Borrow one prepared model connection when a fresh compatible conversation inherits its selection.
// Pattern: Explicit shared resource ownership.
// Usage: submissionSession persists the inherited selection before borrowing its prepared model connection.
// Related: internal/commands/server/submission_session.go, internal/commands/server/lifecycle.go.
package server

import "context"

func (s *Service) transferModel(ctx context.Context, old, next *Conversation) error {
	if next.Models == nil {
		return failure("failed_precondition", "session model persistence is unavailable")
	}
	if err := next.Models.SaveSelection(ctx, next.Runtime.Session.ID, old.Runtime.Selection); err != nil {
		return err
	}
	if old.Model == nil {
		return nil
	}
	borrowed, err := old.Model.Borrow()
	if err != nil {
		return err
	}
	previous := next.Model
	next.Model = borrowed
	previous.Close()
	return nil
}
