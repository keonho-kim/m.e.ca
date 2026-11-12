// Purpose: Measure actual session-usage reads with owned and inactive SQLite conversations.
// Pattern: I/O boundary benchmark with a shared immutable usage ledger.
// Usage: go test ./internal/commands/server -run '^$' -bench BenchmarkSessionUsage -benchmem.
// Related: internal/commands/server/session_usage.go, internal/storage/sqlite/usage.go.
package server

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"path/filepath"
	"testing"
)

func BenchmarkSessionUsage(b *testing.B) {
	ctx := context.Background()
	project := b.TempDir()
	sessions, err := sqlite.OpenSessions(filepath.Join(b.TempDir(), "sessions"))
	if err != nil {
		b.Fatal(err)
	}
	store, err := sessions.Create(ctx, project, project)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()
	id := store.Session().ID
	if err := store.RecordEvent(ctx, id, record.Event{Kind: "usage", Usage: &model.TokenUsage{Input: 17, Output: 9}}); err != nil {
		b.Fatal(err)
	}
	for _, owned := range []bool{true, false} {
		name := "inactive_open"
		if owned {
			name = "owned_lease"
		}
		b.Run(name, func(b *testing.B) {
			s := detailFixture(store, nil)
			defer s.Close()
			s.conversation.Runtime.Session = store.Session()
			if !owned {
				s.conversation = nil
			}
			s.options.Usage = func(ctx context.Context, id string) (model.TokenUsage, error) {
				return sessions.TokenUsage(ctx, project, id)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				value, err := s.SessionUsage(ctx, id)
				if err != nil || value.Input != 17 {
					b.Fatal(value, err)
				}
			}
		})
	}
}
