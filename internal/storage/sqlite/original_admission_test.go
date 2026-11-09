// Purpose: Verify original reads admit selected bytes before decoding their payloads.
// Pattern: SQLite read-boundary regression test.
// Usage: go test ./internal/storage/sqlite -run TestOriginalReadAdmission.
// Related: internal/storage/sqlite/read_budget.go, internal/storage/sqlite/timeline.go.
package sqlite

import (
	"errors"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestOriginalReadAdmissionPrecedesDecoding(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id := s.Session().ID
	if err := s.AppendMessage(t.Context(), id, model.Message{Role: "assistant", Content: strings.Repeat("x", 1<<20)}); err != nil {
		t.Fatal(err)
	}

	if _, err := s.db.Exec("UPDATE payloads SET body=x'00'"); err != nil {
		t.Fatal(err)
	}
	refused := errors.New("original reservation refused")
	reserve := func(bytes int) error {
		if bytes > 64<<10 {
			return refused
		}
		return nil
	}
	if _, _, err := s.TimelineSummaries(t.Context(), id, 0, 1, history.ReadBounds{MaxBytes: 2 << 20, Reserve: reserve}); !errors.Is(err, refused) {
		t.Fatal("detail payload was decoded before admission", err)
	}
}
