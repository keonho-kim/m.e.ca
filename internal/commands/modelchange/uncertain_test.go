// Purpose: Resolve an uncertain selection write by reading its durable value before reporting failure.
// Pattern: Persistence-boundary regression test.
// Usage: go test ./internal/commands/modelchange -run TestCommitReadsUncertainSelection.
// Related: internal/commands/modelchange/commit.go.
package modelchange

import (
	"context"
	"errors"
	"testing"

	"github.com/keonho-kim/meca/internal/provider/connection"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type uncertainSelection struct{ selected connection.Selection }

func (s *uncertainSelection) SaveSelection(_ context.Context, _ string, selected connection.Selection) error {
	s.selected = selected
	return errors.New("commit acknowledgement lost")
}

type rejectedSelection struct{}

func (rejectedSelection) SaveSelection(context.Context, string, connection.Selection) error {
	return sessionstate.ErrForeignSession
}
func (rejectedSelection) Selection(context.Context, string) (connection.Selection, error) {
	return connection.Selection{}, sessionstate.ErrForeignSession
}

func TestRejectedSelectionIsNotUncertain(t *testing.T) {
	result, err := Commit(t.Context(), rejectedSelection{}, commitVault{}, "foreign", connection.Selection{Model: "rejected"}, "", nil)
	if !errors.Is(err, sessionstate.ErrForeignSession) || result.Uncertain {
		t.Fatalf("rejected write reported as uncertain: %+v %v", result, err)
	}
}
func (s *uncertainSelection) Selection(context.Context, string) (connection.Selection, error) {
	return s.selected, nil
}

func TestCommitReadsUncertainSelection(t *testing.T) {
	store := &uncertainSelection{}
	result, err := Commit(t.Context(), store, commitVault{}, "session", connection.Selection{Model: "confirmed"}, "", nil)
	if err != nil || result.Selection.Model != "confirmed" {
		t.Fatalf("durable selection discarded: %+v %v", result, err)
	}
}
