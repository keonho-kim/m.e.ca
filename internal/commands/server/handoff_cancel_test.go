// Purpose: Reproduce cancellation during a detached running session handoff.
// Pattern: Deterministic session ownership regression test.
// Usage: go test -race ./internal/commands/server -run TestRegressionCanceled.
// Related: internal/commands/server/lifecycle.go.
package server

import (
	"context"
	"errors"
	"testing"
)

func TestRegressionCanceledHandoffPreservesRunningSession(t *testing.T) {
	s := detailFixture(&detailHistory{}, nil)
	defer s.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	next := &Conversation{Close: func() error { return nil }}
	next.Runtime.Session.ID = "session_BBBBBBBBBBBBBBBBBBBBBBBBBB"
	closed := false
	next.Close = func() error { closed = true; return nil }
	s.jobs = map[string]*Conversation{next.Runtime.Session.ID: next}
	s.active = map[string]*request{"running": {task: Task{ID: "running", Session: next.Runtime.Session.ID}, conversation: next}}
	s.conversation.Close = func() error { cancel(); return nil }
	_, err := s.Select(ctx, next.Runtime.Session.ID, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation was not reached", err)
	}
	if closed || s.jobs[next.Runtime.Session.ID] == nil {
		t.Fatalf("running session prematurely closed=%v; registry retained=%v; task still active=%v", closed, s.jobs[next.Runtime.Session.ID] != nil, s.active["running"] != nil)
	}
}
