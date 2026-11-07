// Purpose: Verify cursor reads avoid unrelated databases and snapshots expire without retaining handles.
// Pattern: Filesystem fault injection and deterministic snapshot clock tests.
// Usage: go test -race ./internal/storage/sqlite -run TestConversationSnapshot.
// Related: internal/storage/sqlite/conversation_page.go, internal/storage/sqlite/conversation_snapshot.go.
package sqlite

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func TestConversationSnapshotReadsOnlyRequestedPage(t *testing.T) {
	project := t.TempDir()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	for range 7 {
		s, err := catalog.Create(t.Context(), project, project)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
	all, err := catalog.Conversations(t.Context(), project)
	if err != nil {
		t.Fatal(err)
	}
	_, cursor, err := catalog.ConversationPage(t.Context(), project, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	last, err := catalog.Open(t.Context(), project, all[len(all)-1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := last.db.Exec("PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	if err := last.Close(); err != nil {
		t.Fatal(err)
	}
	page, _, err := catalog.ConversationPage(t.Context(), project, cursor, 2)
	if err != nil || len(page) != 2 || page[0].ID != all[2].ID {
		t.Fatal("cursor scanned an unrelated DB", err)
	}
	fresh, err := catalog.Conversations(t.Context(), project)
	if err != nil || len(fresh) != len(all) || fresh[len(fresh)-1].ID != all[len(all)-1].ID || fresh[len(fresh)-1].Unavailable == "" {
		t.Fatal("fresh listing hid an unsupported database", fresh, err)
	}
}

func TestConversationSnapshotExpiryAndCapacity(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	pages := conversationPages{now: func() time.Time { return now }}
	refs := []conversationReference{{id: "reference", updated: now}}
	first, err := pages.put("project", refs)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := pages.put("project", refs); err != nil || again != first {
		t.Fatal("unchanged ordering was duplicated", err)
	}
	for i := range 7 {
		if _, err := pages.put(fmt.Sprint(i), refs); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pages.put("overflow", refs); !errors.Is(err, sessionstate.ErrPageLimit) {
		t.Fatal(err)
	}
	now = now.Add(conversationPageLifetime)
	if _, err := pages.get("project", first); !errors.Is(err, sessionstate.ErrCursor) {
		t.Fatal("expired cursor accepted", err)
	}
	if len(pages.values) != 0 {
		t.Fatal("expired ordering references retained")
	}
}

func BenchmarkConversationPages(b *testing.B) {
	root := b.TempDir()
	catalog, err := OpenSessions(filepath.Join(root, "sessions"))
	if err != nil {
		b.Fatal(err)
	}
	for range 200 {
		s, err := catalog.Create(b.Context(), root, root)
		if err != nil {
			b.Fatal(err)
		}
		if err := s.Close(); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cursor := ""
		seen := 0
		for {
			page, next, err := catalog.ConversationPage(b.Context(), root, cursor, 25)
			if err != nil {
				b.Fatal(err)
			}
			seen += len(page)
			if next == "" {
				break
			}
			cursor = next
		}
		if seen != 200 {
			b.Fatal("missing or duplicated sessions", seen)
		}
	}
}
