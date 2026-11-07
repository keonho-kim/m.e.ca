// Purpose: Reject invalid journal groups before waiting for a database writer.
// Pattern: Persistence admission and atomicity regression tests.
// Usage: go test ./internal/storage/sqlite -run TestJournalPreparation.
// Related: internal/storage/sqlite/commit.go, internal/storage/sqlite/prepare.go.
package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

func TestJournalPreparationRejectsInvalidGroupBeforeWriterAdmission(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tx, err := s.db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	message := model.Message{Role: "user", Content: "valid request"}
	invalid := record.Event{Kind: "usage"}
	err = s.Commit(ctx, s.Session().ID, []record.Entry{{Message: &message}, {Event: &invalid}})
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("invalid group waited for the reserved writer", err)
	}
}
