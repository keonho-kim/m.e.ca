// Purpose: Restore native detail originals beyond the preview budget without altering stored content.
// Pattern: Real SQLite and native detail-lease integration test.
// Usage: go test ./internal/serverapi -run TestLargeOriginal.
// Related: internal/commands/server/detail.go, internal/storage/sqlite/timeline.go.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/commands/server"
	"strings"
	"testing"
)

func TestLargeOriginalSurvivesSmallerPreviewBudget(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	state, err := f.app.Select(t.Context(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("x", server.MaxReadBytes+1024)
	if err := f.store.AppendMessage(t.Context(), state.Session.ID, model.Message{Role: "assistant", Content: text}); err != nil {
		t.Fatal(err)
	}
	preview, _, err := f.app.History(t.Context(), state.Session.ID, 0, 1)
	if err != nil || len(preview) != 1 || !preview[0].Truncated || len(preview[0].Message.Content) != 16<<10 {
		t.Fatal("bounded preview could not read a large original", err)
	}
	read, err := f.app.OpenDetail(t.Context(), server.DetailSource{Session: state.Session.ID, HistorySequence: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer read.Release()
	value, err := read.Load(nil)
	if err != nil || value.History == nil || value.History.Message.Content != text {
		t.Fatal("original detail was lost", err)
	}
}
