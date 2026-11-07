// Purpose: Keep prepared model connections alive until every conversation and execution borrower returns.
// Pattern: Resource ownership regression test.
// Usage: go test ./internal/commands/conversation.
// Related: internal/commands/conversation/model.go, internal/commands/server/model_transfer.go.
package conversation

import (
	"context"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
)

type generator struct{}

func (generator) Generate(context.Context, model.Request, func(model.Event) error) error { return nil }

func TestModelBorrowClosesOnlyAfterLastOwner(t *testing.T) {
	closed := 0
	owner := NewModel(generator{}, func() { closed++ })
	first, err := owner.Borrow()
	if err != nil {
		t.Fatal(err)
	}
	second, err := owner.Borrow()
	if err != nil {
		t.Fatal(err)
	}
	owner.Close()
	owner.Close()
	if _, err := owner.Borrow(); err == nil {
		t.Fatal("closed handle borrowed again")
	}
	first.Close()
	if closed != 0 {
		t.Fatal("live borrower lost its connection")
	}
	second.Close()
	second.Close()
	if closed != 1 {
		t.Fatal("connection cleanup count", closed)
	}
}
