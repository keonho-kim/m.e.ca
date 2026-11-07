// Purpose: Compose deterministic conversation reads and selection persistence for server use-case tests.
// Pattern: Small consumer-owned I/O test doubles.
// Usage: Lifecycle, settings and read-lease tests use detailFixture without SQL or network calls.
// Related: internal/commands/server/contracts.go, internal/commands/server/projection_test.go.
package server

import (
	"context"
	"sync/atomic"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/history"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type detailHistory struct {
	value record.Event
	reads atomic.Int64
}

func (h *detailHistory) ToolEvent(context.Context, record.EventReference, history.ReadBounds) (record.Event, error) {
	h.reads.Add(1)
	return h.value, nil
}
func (*detailHistory) TimelineSummaries(context.Context, string, int64, int, history.ReadBounds) ([]history.Record, int64, error) {
	return nil, 0, nil
}
func (*detailHistory) SetTeamMode(context.Context, bool) error { return nil }

type selectionStore struct {
	selected connection.Selection
	err      error
}

func (s *selectionStore) SaveSelection(_ context.Context, _ string, value connection.Selection) error {
	if s.err != nil {
		return s.err
	}
	s.selected = value
	return nil
}

func detailFixture(h History, vault *credentials.Vault) *Service {
	s, _ := projectionFixture()
	s.active = nil
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.closeDone = make(chan struct{})
	s.conversation = &Conversation{Runtime: agentrequest.Runtime{Vault: vault, Session: sessionstate.Session{ID: "session_AAAAAAAAAAAAAAAAAAAAAAAAAA"}}, History: h, Models: &selectionStore{}, Close: func() error { return nil }}
	return s
}

func (s *selectionStore) Selection(context.Context, string) (connection.Selection, error) {
	return s.selected, nil
}
