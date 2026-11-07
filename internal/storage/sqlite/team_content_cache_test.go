// Purpose: Verify shared-text identity reuse, bounded retention and last-reader cleanup.
// Pattern: Pure cache and temporary SQLite lifecycle tests.
// Usage: Team context tests cover concurrent readers and immutable identity invalidation.
// Related: internal/storage/sqlite/team_content_cache.go, internal/storage/sqlite/team_sessions.go.
package sqlite

import (
	"context"
	"strings"
	"sync"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestHistoryReaderDoesNotReleaseActiveTeamCacheLease(t *testing.T) {
	root, child := teamStorage(t)
	if _, err := root.teamContent.encoding(1, "active shared context"); err != nil {
		t.Fatal(err)
	}
	page := teamPage{parent: root}
	if _, err := page.source(context.Background(), child.session.ID); err != nil {
		t.Fatal(err)
	}
	if err := page.close(); err != nil {
		t.Fatal(err)
	}
	if root.teamContent.users != 1 || len(root.teamContent.values) != 1 {
		t.Fatal("history reader released an active session's cache ownership")
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
	if root.teamContent.users != 0 || root.teamContent.values != nil {
		t.Fatal("last active session retained its cache lease")
	}
}

func TestTeamTextCacheReleasesAfterLastPrivateReader(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	member, err := root.CreateTeamAgent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := root.OpenTeamAgent(ctx, member.Session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { second.Close() })
	value, err := root.AppendTeamMessage(ctx, channel.Message{From: "Hana", Text: strings.Repeat("shared result ", 10000)})
	if err != nil {
		t.Fatal(err)
	}
	message := value.Context()
	for _, reader := range []*Store{child, second} {
		if err := reader.CommitContext(ctx, reader.session.ID, []runrecord.Entry{{Message: &message}}, value.Sequence); err != nil {
			t.Fatal(err)
		}
	}
	if root.teamContent.users != 2 || len(root.teamContent.values) != 1 {
		t.Fatal("private readers did not share one identity")
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
	if err := child.Close(); err != nil {
		t.Fatal(err)
	}
	if root.teamContent.users != 1 || len(root.teamContent.values) != 1 {
		t.Fatal("closing one reader released another reader's cache")
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if root.teamContent.users != 0 || root.teamContent.values != nil || root.teamContent.order != nil || root.teamContent.bytes != 0 {
		t.Fatal("last reader retained shared context")
	}
}

func TestTeamTextCacheBoundsAndContentReplacement(t *testing.T) {
	var cache teamContentCache
	cache.acquire()
	first, _ := cache.encoding(1, strings.Repeat("a", 4096))
	replaced, _ := cache.encoding(1, strings.Repeat("b", 4096))
	if first.id == replaced.id {
		t.Fatal("replaced content reused the earlier digest")
	}
	var workers sync.WaitGroup
	for i := range 200 {
		workers.Go(func() {
			if _, err := cache.encoding(int64(i+2), strings.Repeat("x", 16384)); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	if len(cache.values) > teamContentEntries || cache.bytes > teamContentBytes {
		t.Fatal("shared text exceeded its retention budget")
	}
	if _, err := cache.encoding(999, strings.Repeat("large", teamContentBytes)); err != nil {
		t.Fatal(err)
	}
	if _, retained := cache.values[999]; retained {
		t.Fatal("oversized accepted content stayed in the cache")
	}
	cache.release()
	if cache.values != nil {
		t.Fatal("released cache retained content")
	}
}

func TestSanitizedLocalPeerContentDoesNotDependOnPublicOriginal(t *testing.T) {
	root, child := teamStorage(t)
	ctx := context.Background()
	value, err := root.AppendTeamMessage(ctx, channel.Message{From: "Hana", Text: strings.Repeat("private-original ", 1000)})
	if err != nil {
		t.Fatal(err)
	}
	message := value.Context()
	message.Content = strings.ReplaceAll(message.Content, "private-original", "[redacted]")
	message.ID += "#local"
	local := *message.Source
	local.Sequence = 0
	message.Source = &local
	if err := child.CommitContext(ctx, child.session.ID, []runrecord.Entry{{Message: &message}}, value.Sequence); err != nil {
		t.Fatal(err)
	}
	if _, err := root.db.Exec("DELETE FROM tool_events WHERE kind='team_message'"); err != nil {
		t.Fatal(err)
	}
	got, err := child.ContextMessages(ctx, child.session.ID, 0, 16<<20, false)
	if err != nil || len(got) != 1 || got[0].Content != message.Content {
		t.Fatal("local sanitized text attempted to restore different shared bytes", got, err)
	}
}

func BenchmarkTeamTextIdentity(b *testing.B) {
	text := strings.Repeat("한글 shared \"body\" <tag>\n", 32768)
	for _, shared := range []bool{false, true} {
		name := "uncached"
		if shared {
			name = "shared"
		}
		b.Run(name, func(b *testing.B) {
			var cache teamContentCache
			cache.acquire()
			defer cache.release()
			cache.encoding(1, text)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				var err error
				if shared {
					_, err = cache.encoding(1, text)
				} else {
					_, err = encodeTeamText(text)
				}
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
