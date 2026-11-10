// Purpose: Enforce detail authorization and cancellation while lending immutable source records.
// Pattern: Conversation read-lease lifecycle tests.
// Usage: go test -race ./internal/commands/server -run TestDetail.
// Related: internal/commands/server/detail.go, internal/commands/server/operations.go.
package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/config/credentials"
)

func TestDetailLeaseReturnsOriginalAndReleasesSession(t *testing.T) {
	s := detailFixture(&detailHistory{value: record.Event{Kind: "tool_succeeded", Text: "original"}}, &credentials.Vault{})
	defer s.Close()
	read, err := s.OpenDetail(t.Context(), DetailSource{Tool: record.EventReference{Sequence: 1}})
	if err != nil {
		t.Fatal(err)
	}
	defer read.Release()
	original, err := read.Load(nil)
	if err != nil || original.Tool.Text != "original" || read.Source.Tool.Session != s.State().Session.ID {
		t.Fatal(original, err)
	}
	read.Release()
	if _, err := read.Load(nil); !errors.Is(err, context.Canceled) {
		t.Fatal("released lease remained usable", err)
	}
}

func TestDetailRejectsResolvedApproval(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	s.active = map[string]*request{"approval-task": {conversation: s.conversation, approvals: map[string]*pendingApproval{"approve": {value: tool.Approval{Preview: "proposal"}}}}}
	read, err := s.OpenDetail(t.Context(), DetailSource{ApprovalID: "approve"})
	if err != nil {
		t.Fatal(err)
	}
	read.Release()
	delete(s.active["approval-task"].approvals, "approve")
	if _, err := s.OpenDetail(t.Context(), DetailSource{ApprovalID: "approve"}); err == nil {
		t.Fatal("resolved approval remained readable")
	}
}

func TestDetailLeaseCanceledBeforeSessionCleanup(t *testing.T) {
	s := detailFixture(&detailHistory{}, &credentials.Vault{})
	defer s.Close()
	read, err := s.OpenDetail(t.Context(), DetailSource{Tool: record.EventReference{Sequence: 1}})
	if err != nil {
		t.Fatal(err)
	}
	defer read.Release()
	closed := make(chan error, 1)
	go func() { closed <- s.CloseSession(t.Context(), s.State().Session.ID) }()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	select {
	case <-read.Context.Done():
	case <-ctx.Done():
		t.Error("session close did not cancel its detail reader")
	}
	read.Release()
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	if s.State().Session.ID != "" {
		t.Fatal("closed session remained selected")
	}
}
