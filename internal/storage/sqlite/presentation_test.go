// Purpose: Preserve display positions through full and bounded history restoration.
// Pattern: SQLite round-trip contract test.
// Usage: go test ./internal/storage/sqlite -run TestPresentation.
// Related: internal/storage/history/display.go, internal/serverapi/history_content.go.
package sqlite

import (
	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/history"
	"testing"
)

func TestPresentationSurvivesHistoryProjection(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id := "request.00000000000000000001"
	if err := s.Commit(t.Context(), s.Session().ID, []record.Entry{
		{Event: &record.Event{Kind: "thinking", Thinking: "Reason", Presentation: id}},
		{Message: &model.Message{Role: "assistant", Content: "Answer", Presentation: id}},
	}); err != nil {
		t.Fatal(err)
	}
	for _, bounds := range []history.ReadBounds{{}, {MaxBytes: 16384, TextBytes: 4096}} {
		rows, _, err := s.TimelineSummaries(t.Context(), s.Session().ID, 0, 20, bounds)
		if err != nil || len(rows) != 2 {
			t.Fatalf("history: %+v %v", rows, err)
		}
		if rows[0].Event.Presentation != id || rows[1].Message.Presentation != id {
			t.Fatalf("lost display identity: %+v", rows)
		}
	}
}
