// Purpose: Expose current state and visible history without retaining a read lock during storage I/O.
// Pattern: Server read use cases.
// Usage: State and reconnects use snapshots; clients page durable history and requested originals.
// Related: internal/commands/server/events.go, internal/storage/sqlite/timeline.go.
package server

import (
	"context"
	"sort"
	"time"

	"github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/history"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (s *Service) State() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot()
}

func (s *Service) snapshot() Snapshot { return s.snapshotPage("", "", 32) }

func (s *Service) snapshotPage(taskID, after string, limit int) Snapshot {
	value := s.baseSnapshot()
	var ids []string
	if taskID != "" {
		if r := s.active[taskID]; r != nil && taskID > after {
			ids = []string{taskID}
		}
	} else {
		ids = make([]string, 0, len(s.active))
		for id := range s.active {
			if id > after {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
	}
	value.TotalTasks = len(s.active)
	if len(ids) > limit {
		value.NextTask = ids[limit-1]
		ids = ids[:limit]
	}
	draftBudget, activityBudget := 512<<10, 256<<10
	for _, id := range ids {
		r := s.active[id]
		value.Tasks = append(value.Tasks, r.task)
		if c := r.conversation; c != nil {
			value.Executions = append(value.Executions, ExecutionState{Task: r.task.ID, Session: c.Runtime.Session, Selection: c.Runtime.Selection, Team: r.teamEnabled, StartedAt: r.started, Statistics: r.statistics()})
		}
		for id, pending := range r.approvals {
			e := ProjectEvent(Event{Record: record.Event{Approval: &pending.value}})
			var expires time.Time
			if pending.ctx != nil {
				expires, _ = pending.ctx.Deadline()
			}
			value.Approvals = append(value.Approvals, Approval{ID: id, Task: r.task.ID, Value: e.Record, Truncated: e.Truncated, ExpiresAt: expires})
		}
		for _, draft := range r.drafts {
			d := draft.Draft
			d.Task = r.task.ID
			d.TextBytes, d.ThinkingBytes = int64(draft.textBytes), int64(draft.thinkingBytes)
			d.Text, d.Thinking = prefix(d.Text, min(4096, draftBudget)), prefix(d.Thinking, min(4096, max(0, draftBudget-len(d.Text))))
			draftBudget -= len(d.Text) + len(d.Thinking)
			d.Truncated = d.Truncated || int64(len(d.Text)) < d.TextBytes || int64(len(d.Thinking)) < d.ThinkingBytes
			value.Drafts = append(value.Drafts, d)
		}
		for _, activity := range r.activity {
			activity.Workspace = s.options.WorkspaceID
			if size := eventBytes(activity); size <= activityBudget {
				value.Activity = append(value.Activity, activity)
				activityBudget -= size
			} else {
				value.ActivityTruncated = true
			}
		}
	}
	sort.Slice(value.Tasks, func(i, j int) bool { return value.Tasks[i].ID < value.Tasks[j].ID })
	if c := s.conversation; c != nil {
		value.Session, value.Selection, value.Team = c.Runtime.Session, c.Runtime.Selection, c.TeamEnabled
	}
	if s.last != nil {
		task := *s.last
		value.Task = &task
	}
	var r *request
	if taskID == "" {
		r = s.displayRequest()
	}
	if taskID != "" {
		r = s.active[taskID]
		value.Task, value.Execution = nil, nil
		value.Session, value.Selection, value.Team = sessionstate.Session{}, connection.Selection{}, false
		if s.last != nil && s.last.ID == taskID {
			copy := *s.last
			value.Task = &copy
		}
	}
	if r != nil {
		if c := r.conversation; c != nil {
			if taskID != "" {
				value.Session, value.Selection, value.Team = c.Runtime.Session, c.Runtime.Selection, c.TeamEnabled
			}
			value.Execution = &ExecutionState{Task: r.task.ID, Session: c.Runtime.Session, Selection: c.Runtime.Selection, Team: r.teamEnabled, StartedAt: r.started, Statistics: r.statistics()}
		}
		task := r.task
		value.Task = &task
	}
	sort.Slice(value.Approvals, func(i, j int) bool { return value.Approvals[i].ID < value.Approvals[j].ID })
	sort.Slice(value.Drafts, func(i, j int) bool {
		return value.Drafts[i].Task+value.Drafts[i].Agent < value.Drafts[j].Task+value.Drafts[j].Agent
	})
	sort.Slice(value.Executions, func(i, j int) bool { return value.Executions[i].Task < value.Executions[j].Task })
	return value
}

func (s *Service) History(ctx context.Context, session string, before int64, limit int) ([]history.Record, int64, error) {
	if before < 0 || limit < 1 || limit > 100 || session != "" && !sessionstate.ValidID(session) {
		return nil, 0, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerHistoryLimitMustBeToAndCursor)
	}
	var c *Conversation
	var done func()
	var err error
	if session != "" {
		c, ctx, done, err = s.beginSessionRead(ctx, session)
	} else {
		c, ctx, done, err = s.beginRead(ctx)
	}
	if err != nil {
		return nil, 0, err
	}
	defer done()
	return c.History.TimelineSummaries(ctx, c.Runtime.Session.ID, before, limit, history.ReadBounds{MaxBytes: MaxReadBytes, TextBytes: 16 << 10})
}
