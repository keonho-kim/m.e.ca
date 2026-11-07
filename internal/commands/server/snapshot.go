// Purpose: Expose current state and visible history without retaining a read lock during storage I/O.
// Pattern: Server read use cases.
// Usage: State and reconnects use snapshots; clients page durable history and requested originals.
// Related: internal/commands/server/events.go, internal/storage/sqlite/timeline.go.
package server

import (
	"context"
	"sort"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (s *Service) State() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot()
}

func (s *Service) snapshot() Snapshot {
	permission, _ := s.options.Config.Permission.MarshalText()
	value := Snapshot{Cursor: s.feed.cursor(), Permission: string(permission)}
	if c := s.conversation; c != nil {
		value.Session, value.Selection, value.Team = c.Runtime.Session, c.Runtime.Selection, c.TeamEnabled
	}
	if s.last != nil {
		task := *s.last
		value.Task = &task
	}
	if r := s.active; r != nil {
		if c := r.conversation; c != nil {
			value.Execution = &ExecutionState{Session: c.Runtime.Session, Selection: c.Runtime.Selection, Team: c.TeamEnabled}
		}
		task := r.task
		value.Task = &task
		for id, pending := range r.approvals {
			e := ProjectEvent(Event{Record: record.Event{Approval: &pending.value}})
			value.Approvals = append(value.Approvals, Approval{ID: id, Value: e.Record, Truncated: e.Truncated})
		}
		for _, draft := range r.drafts {
			value.Drafts = append(value.Drafts, draft.Draft)
		}
		sort.Slice(value.Approvals, func(i, j int) bool { return value.Approvals[i].ID < value.Approvals[j].ID })
		sort.Slice(value.Drafts, func(i, j int) bool { return value.Drafts[i].Agent < value.Drafts[j].Agent })
	}
	return value
}

func (s *Service) History(ctx context.Context, before int64, limit int) ([]history.Record, int64, error) {
	if before < 0 || limit < 1 || limit > 100 {
		return nil, 0, failure("invalid_argument", "history limit must be 1 to 100 and cursor must be nonnegative")
	}
	c, ctx, done, err := s.beginRead(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer done()
	return c.History.TimelineSummaries(ctx, c.Runtime.Session.ID, before, limit, history.ReadBounds{MaxBytes: MaxReadBytes, TextBytes: 16 << 10})
}
