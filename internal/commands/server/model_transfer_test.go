// Purpose: Verify inherited model clients keep exactly one cleanup owner after session replacement.
// Pattern: Resource ownership regression test.
// Usage: go test ./internal/commands/server -run TestTransferModel.
// Related: internal/commands/server/model_transfer.go.
package server

import (
	"context"
	"errors"
	conversationruntime "github.com/keonho-kim/meca/internal/commands/conversation"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
)

type transferGenerator struct{}

func (*transferGenerator) Generate(context.Context, model.Request, func(model.Event) error) error {
	return nil
}

func TestTransferModelPreservesOwnershipOnFailureAndSuccess(t *testing.T) {
	s := detailFixture(&detailHistory{}, nil)
	defer s.Close()
	old := s.conversation
	closed := 0
	old.Model = conversationruntime.NewModel(&transferGenerator{}, func() { closed++ })
	next := &Conversation{Models: &selectionStore{err: errors.New("disk failure")}, Close: func() error { return nil }}
	if err := s.transferModel(t.Context(), old, next); err == nil || old.Model == nil || next.Model != nil {
		t.Fatal("failed transfer moved ownership", err)
	}
	next.Models = &selectionStore{}
	if err := s.transferModel(t.Context(), old, next); err != nil {
		t.Fatal(err)
	}
	if old.Model == nil || next.Model == nil || old.Model.Generator() != next.Model.Generator() {
		t.Fatal("client ownership was not transferred")
	}
	if err := closeConversation(old); err != nil {
		t.Fatal(err)
	}
	if closed != 0 {
		t.Fatal("old session closed inherited model")
	}
	if err := closeConversation(next); err != nil {
		t.Fatal(err)
	}
	if closed != 1 {
		t.Fatal("prepared model cleanup count", closed)
	}
}
