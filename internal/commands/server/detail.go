// Purpose: Authorize one visible detail source and lend its conversation until the reader releases it.
// Pattern: Consumer-owned read lease independent of serialization and page storage.
// Usage: serverapi/detail loads an immutable original only when its wire cache needs refreshing.
// Related: internal/commands/server/operations.go, internal/storage/history/record.go.
package server

import (
	"context"
	"errors"
	"sync"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/history"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type DetailSource struct {
	Tool            record.EventReference
	ApprovalID      string
	HistorySequence int64
}

type DetailOriginal struct {
	Tool     *record.Event
	History  *history.Record
	Approval *tool.Approval
}

// DetailRead pins the selected conversation. The caller releases it exactly once.
// Load must not be used after Release; Context is canceled when that session closes.
type DetailRead struct {
	Context context.Context
	Session string
	Source  DetailSource
	Load    func() (DetailOriginal, error)
	Release func()
}

func (s *Service) OpenDetail(ctx context.Context, source DetailSource) (DetailRead, error) {
	choices := 0
	if source.Tool.Sequence != 0 {
		choices++
	}
	if source.ApprovalID != "" {
		choices++
	}
	if source.HistorySequence != 0 {
		choices++
	}
	if choices != 1 || source.Tool.Sequence < 0 || source.HistorySequence < 0 || source.HistorySequence == 1<<63-1 || source.Tool.Session != "" && !sessionstate.ValidID(source.Tool.Session) {
		return DetailRead{}, failure("invalid_argument", "choose exactly one valid detail reference")
	}
	c, ctx, done, err := s.beginRead(ctx)
	if err != nil {
		return DetailRead{}, err
	}
	var pending *pendingApproval
	if source.ApprovalID != "" {
		s.mu.Lock()
		if s.active != nil {
			pending = s.active.approvals[source.ApprovalID]
		}
		s.mu.Unlock()
		if pending == nil {
			done()
			return DetailRead{}, failure("conflict", "approval is no longer pending")
		}
	}
	if source.Tool.Sequence != 0 && source.Tool.Session == "" {
		source.Tool.Session = c.Runtime.Session.ID
	}
	return DetailRead{Context: ctx, Session: c.Runtime.Session.ID, Source: source, Release: sync.OnceFunc(done), Load: func() (DetailOriginal, error) {
		if err := ctx.Err(); err != nil {
			return DetailOriginal{}, err
		}
		return c.loadDetail(ctx, source, pending)
	}}, nil
}

func (c *Conversation) loadDetail(ctx context.Context, source DetailSource, pending *pendingApproval) (DetailOriginal, error) {
	if pending != nil {
		return DetailOriginal{Approval: &pending.value}, nil
	}
	if source.Tool.Sequence != 0 {
		value, err := c.History.ToolEvent(ctx, source.Tool, history.ReadBounds{MaxBytes: MaxReadBytes})
		if errors.Is(err, history.ErrNotFound) || errors.Is(err, sessionstate.ErrNotFound) {
			return DetailOriginal{}, failure("not_found", "tool detail was not found")
		}
		return DetailOriginal{Tool: &value}, err
	}
	values, _, err := c.History.TimelineSummaries(ctx, c.Runtime.Session.ID, source.HistorySequence+1, 1, history.ReadBounds{MaxBytes: MaxReadBytes})
	if err != nil {
		return DetailOriginal{}, err
	}
	if len(values) != 1 || values[0].Sequence != source.HistorySequence {
		return DetailOriginal{}, failure("not_found", "history record was not found")
	}
	return DetailOriginal{History: &values[0]}, nil
}
