// Purpose: Verify ordered bounded session pages and project-bound cursor validation.
// Pattern: Temporary catalog pagination contract tests.
// Usage: go test ./internal/storage/sqlite -run TestConversationPage.
// Related: internal/storage/sqlite/conversation_page.go, api/meca/v1/state.proto.
package sqlite

import (
	"path/filepath"
	"testing"
)

func TestConversationPageOrderingAndIsolation(t *testing.T) {
	sessions, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	for range 7 {
		s, err := sessions.Create(t.Context(), project, project)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
	all, err := sessions.Conversations(t.Context(), project)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	cursor := ""
	for {
		page, next, err := sessions.ConversationPage(t.Context(), project, cursor, 2)
		if err != nil || len(page) > 2 {
			t.Fatal(err)
		}
		for _, value := range page {
			ids = append(ids, value.ID)
		}
		if next == "" {
			break
		}
		if cursor == "" {
			if _, _, err := sessions.ConversationPage(t.Context(), t.TempDir(), next, 2); err == nil {
				t.Fatal("foreign project cursor accepted")
			}
		}
		cursor = next
	}
	if len(ids) != len(all) {
		t.Fatal("missing or duplicated sessions", ids)
	}
	for i, id := range ids {
		if id != all[i].ID {
			t.Fatal("page ordering changed")
		}
	}
	if _, _, err := sessions.ConversationPage(t.Context(), project, "invalid", 2); err == nil {
		t.Fatal("invalid cursor accepted")
	}
}
