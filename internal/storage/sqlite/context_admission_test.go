// Purpose: Reject context hydration before reading payload bodies when process admission fails.
// Pattern: Corrupted-body fault injection at the repository boundary.
// Usage: go test ./internal/storage/sqlite -run TestContextAdmissionPrecedesHydration.
// Related: internal/storage/sqlite/context.go, internal/storage/sqlite/context_size.go.
package sqlite

import (
	"errors"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
)

func TestContextAdmissionPrecedesHydration(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := s.AppendMessage(t.Context(), s.Session().ID, model.Message{Role: "user", Content: strings.Repeat("payload", 20<<10)}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("UPDATE payloads SET body=x'00'"); err != nil {
		t.Fatal(err)
	}
	denied := errors.New("context memory admission denied")
	reserved := 0
	values, err := s.ContextMessages(t.Context(), s.Session().ID, 0, 1<<20, true, func(bytes int) error { reserved = bytes; return denied })
	if !errors.Is(err, denied) || len(values) != 0 || reserved < 100<<10 {
		t.Fatal("payload restoration preceded admission", len(values), reserved, err)
	}
}
