// Purpose: Read explicit bounded state pages independently of presentation selection.
// Pattern: Application read boundary and deterministic projection limits.
// Usage: State commands and transport encoders share page and preview budgets.
// Related: internal/commands/server/snapshot.go, internal/serverapi/projection.go.
package server

import (
	"context"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"slices"
)

func (s *Service) StatePage(ctx context.Context, session, taskID, after string, limit int) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if limit == 0 {
		limit = 32
	}
	if limit < 1 || limit > 100 || len(taskID) > 128 || len(after) > 128 {
		return Snapshot{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	if session == "" {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.snapshotPage(taskID, after, limit), nil
	}
	if taskID != "" {
		task, err := s.TaskStatus(ctx, taskID)
		if err != nil {
			return Snapshot{}, err
		}
		if task.Session != session {
			return Snapshot{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerTaskSessionMismatch)
		}
	}
	c, _, done, err := s.beginSessionRead(ctx, session)
	if err != nil {
		return Snapshot{}, err
	}
	defer done()
	s.mu.Lock()
	defer s.mu.Unlock()
	value := s.sessionSnapshot(c)
	if taskID != "" {
		value = s.snapshotPage(taskID, "", 1)
		value.Session, value.Selection, value.Team = c.Runtime.Session, c.Runtime.Selection, c.TeamEnabled
		value.TotalTasks = len(value.Tasks)
		value.NextTask = ""
	}
	return value, nil
}

// BoundSnapshot also protects encoder callers that supply a manually assembled snapshot.
func BoundSnapshot(value Snapshot) Snapshot {
	if len(value.Tasks) > 100 {
		value.NextTask = value.Tasks[99].ID
		value.Tasks = value.Tasks[:100]
	}
	if len(value.Executions) > 100 {
		value.Executions = value.Executions[:100]
	}
	value.Drafts = slices.Clone(value.Drafts)
	budget := 512 << 10
	for i := range value.Drafts {
		d := &value.Drafts[i]
		text, thinking := prefix(d.Text, min(4096, budget)), prefix(d.Thinking, min(4096, max(0, budget-min(4096, len(d.Text)))))
		d.Truncated = d.Truncated || len(text) < len(d.Text) || len(thinking) < len(d.Thinking)
		d.Text, d.Thinking = text, thinking
		budget -= len(text) + len(thinking)
	}
	return value
}
