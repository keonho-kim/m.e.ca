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
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/history"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type DetailView uint8

const (
	DetailDocument DetailView = iota
	DetailToolRequest
	DetailToolResult
	DetailText
	DetailThinking
)

type DetailSource struct {
	View            DetailView
	Session         string
	Tool            record.EventReference
	ApprovalID      string
	HistorySequence int64
	Field           string
	ListFields      bool
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
	Load    func(func(int) error) (DetailOriginal, error)
	Release func()
}

func (s *Service) OpenDetail(ctx context.Context, source DetailSource) (DetailRead, error) {
	if (source.Field != "" || source.ListFields) && source.View != DetailToolRequest && source.View != DetailToolResult || source.ListFields && source.Field != "" || len(source.Field) > 512 {
		return DetailRead{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerChooseExactlyOneValidDetailReference)
	}
	if (source.View == DetailToolRequest || source.View == DetailToolResult) && source.Tool.Sequence == 0 || (source.View == DetailText || source.View == DetailThinking) && source.HistorySequence == 0 {
		return DetailRead{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerChooseExactlyOneValidDetailReference)
	}
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
	if source.View > DetailThinking || source.View != DetailDocument && source.ApprovalID != "" || choices != 1 || source.Tool.Sequence < 0 || source.HistorySequence < 0 || source.HistorySequence == 1<<63-1 || source.Tool.Session != "" && !sessionstate.ValidID(source.Tool.Session) || source.Session != "" && !sessionstate.ValidID(source.Session) {
		return DetailRead{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerChooseExactlyOneValidDetailReference)
	}
	c, ctx, done, pending, err := s.beginDetailRead(ctx, source)
	if err != nil {
		return DetailRead{}, err
	}
	if source.Tool.Sequence != 0 && source.Tool.Session == "" {
		source.Tool.Session = c.Runtime.Session.ID
	}
	return DetailRead{Context: ctx, Session: c.Runtime.Session.ID, Source: source, Release: sync.OnceFunc(done), Load: func(reserve func(int) error) (DetailOriginal, error) {
		if err := ctx.Err(); err != nil {
			return DetailOriginal{}, err
		}
		return c.loadDetail(ctx, source, pending, reserve)
	}}, nil
}

func (s *Service) beginDetailRead(ctx context.Context, source DetailSource) (*Conversation, context.Context, func(), *pendingApproval, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, nil, nil, nil, faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerIsShuttingDown)
	}
	c := s.conversation
	if source.Session != "" {
		c = s.ownedConversation(source.Session)
	}
	var pending *pendingApproval
	if source.ApprovalID != "" {
		for _, r := range s.active {
			if value := r.approvals[source.ApprovalID]; value != nil && (source.Session == "" || r.task.Session == source.Session) {
				c, pending = r.conversation, value
				break
			}
		}
		if pending == nil {
			s.mu.Unlock()
			return nil, nil, nil, nil, faults.New(diagcatalog.Conflict, diagcatalog.ServerApprovalIsNoLongerPending)
		}
	}
	if pending == nil && source.Session != "" {
		s.mu.Unlock()
		c, ctx, done, err := s.beginSessionRead(ctx, source.Session)
		return c, ctx, done, nil, err
	}
	if c == nil {
		s.mu.Unlock()
		return nil, nil, nil, nil, faults.New(diagcatalog.NotFound, diagcatalog.ServerConversationIsNotCurrentlyOwned)
	}
	c, ctx, done, err := s.borrowReadLocked(ctx, c)
	s.mu.Unlock()
	return c, ctx, done, pending, err
}

func (c *Conversation) loadDetail(ctx context.Context, source DetailSource, pending *pendingApproval, reserve func(int) error) (DetailOriginal, error) {
	if pending != nil {
		return DetailOriginal{Approval: &pending.value}, nil
	}
	if source.Tool.Sequence != 0 {
		var value record.Event
		var err error
		bounds := history.ReadBounds{MaxBytes: MaxOriginalBytes, Reserve: reserve}
		if source.View == DetailToolRequest {
			reader, ok := c.History.(interface {
				ToolRequest(context.Context, record.EventReference, history.ReadBounds) (record.Event, error)
			})
			if !ok {
				return DetailOriginal{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerWorkspaceUnavailable)
			}
			value, err = reader.ToolRequest(ctx, source.Tool, bounds)
		} else {
			value, err = c.History.ToolEvent(ctx, source.Tool, bounds)
		}
		if errors.Is(err, history.ErrNotFound) || errors.Is(err, sessionstate.ErrNotFound) {
			return DetailOriginal{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerToolDetailWasNotFound)
		}
		return DetailOriginal{Tool: &value}, err
	}
	values, _, err := c.History.TimelineSummaries(ctx, c.Runtime.Session.ID, source.HistorySequence+1, 1, history.ReadBounds{MaxBytes: MaxOriginalBytes, Reserve: reserve})
	if err != nil {
		return DetailOriginal{}, err
	}
	if len(values) != 1 || values[0].Sequence != source.HistorySequence {
		return DetailOriginal{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerHistoryRecordWasNotFound)
	}
	return DetailOriginal{History: &values[0]}, nil
}
