// Purpose: Leave missing conversation titles to each presentation locale without rewriting real titles.
// Pattern: Repository-to-presentation contract regression.
// Usage: go test ./internal/storage/sqlite -run TestUntitledConversation.
// Related: internal/storage/sqlite/session.go, internal/tui/startup/resume.go.
package sqlite

import (
	"github.com/keonho-kim/meca/internal/agent/model"
	"testing"
)

func TestUntitledConversationHasNoStoredEnglishFallback(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c, err := readConversation(t.Context(), s.db, s.session.ID)
	if err != nil || c.Summary != "" {
		t.Fatal(c.Summary, err)
	}
	if err := s.AppendMessage(t.Context(), s.session.ID, model.Message{Role: "user", Content: "New conversation"}); err != nil {
		t.Fatal(err)
	}
	c, err = readConversation(t.Context(), s.db, s.session.ID)
	if err != nil || c.Summary != "New conversation" {
		t.Fatal("rewrote an actual title", c.Summary, err)
	}
}
