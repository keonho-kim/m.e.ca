// Purpose: Keep transcript watermarks reachable when hidden model-context records follow visible output.
// Pattern: SQLite read-boundary regression.
// Usage: go test ./internal/storage/sqlite -run TestTimelineHead.
// Related: internal/storage/sqlite/display_page.go, internal/storage/sqlite/timeline.go.
package sqlite

import (
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestTimelineHeadMatchesVisibleHistory(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, session := t.Context(), s.Session().ID
	if head, err := s.TimelineHead(ctx, session); err != nil || head != 0 {
		t.Fatalf("empty history boundary: %d, %v", head, err)
	}
	for _, role := range []string{"user", "system"} {
		if err := s.AppendMessage(ctx, session, model.Message{Role: role, Content: role + " content"}); err != nil {
			t.Fatal(err)
		}
	}
	rows, _, err := s.TimelineSummaries(ctx, session, 0, 32, history.ReadBounds{})
	if err != nil {
		t.Fatal(err)
	}
	head, err := s.TimelineHead(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || head != rows[0].Sequence {
		t.Fatalf("unreachable watermark %d: %v", head, rows)
	}
}
