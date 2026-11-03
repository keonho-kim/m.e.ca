// Purpose: Measure shared compaction restoration and interleaved historical private envelopes.
// Pattern: Fixed-history storage benchmarks.
// Usage: go test ./internal/storage/sqlite -run '^$' -bench 'BenchmarkTeam(SharedMemory|HistoricalReaders)' -benchmem.
// Related: internal/storage/sqlite/memory_sources.go, internal/storage/sqlite/team_reference_page.go.
package sqlite

import (
	"context"
	"fmt"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	teamchannel "github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func BenchmarkTeamSharedMemory(b *testing.B) {
	root := benchmarkPayloadStore(b)
	ctx := context.Background()
	member, err := root.CreateTeamAgent(ctx)
	if err != nil {
		b.Fatal(err)
	}
	child, err := root.OpenTeamAgent(ctx, member.Session)
	if err != nil {
		b.Fatal(err)
	}
	defer child.Close()
	for i := range 32 {
		value, err := root.AppendTeamMessage(ctx, teamchannel.Message{From: "Hana", Text: strings.Repeat("Observed file content.\n", 3000) + fmt.Sprint(i)})
		if err != nil {
			b.Fatal(err)
		}
		message := value.Context()
		if err := child.CommitContext(ctx, child.session.ID, []runrecord.Entry{{Message: &message}}, value.Sequence); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := child.MemoryRecords(ctx, child.session.ID, 0, 1<<62)
		if err != nil || len(out) != 32 {
			b.Fatal(err)
		}
	}
}

func BenchmarkTeamHistoricalReaders(b *testing.B) {
	for _, count := range []int{2, 8, 16} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			root := benchmarkPayloadStore(b)
			ctx := context.Background()
			refs := make([]runrecord.EventReference, count)
			for i := range count {
				member, err := root.CreateTeamAgent(ctx)
				if err != nil {
					b.Fatal(err)
				}
				child, err := root.OpenTeamAgent(ctx, member.Session)
				if err != nil {
					b.Fatal(err)
				}
				if err := child.RecordEvent(ctx, child.session.ID, runrecord.Event{Kind: "tool_succeeded", CallID: "file", Call: &toolapi.Call{ID: "file", Name: "write_file", Arguments: []byte(`{"path":"a.txt","content":"text"}`)}, Text: "file created", Output: []byte(`{"changed":true}`)}); err != nil {
					b.Fatal(err)
				}
				refs[i] = runrecord.EventReference{Session: child.session.ID, Sequence: child.latestEvent.Load()}
				if err := child.Close(); err != nil {
					b.Fatal(err)
				}
			}
			for i := range 64 {
				if err := root.RecordEvent(ctx, root.session.ID, runrecord.Event{Kind: "team_reference", Agent: identity.Name(i % count), Reference: &refs[i%count]}); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rows, _, err := root.TimelineSummaries(ctx, root.session.ID, 0, 64)
				if err != nil || len(rows) != 64 {
					b.Fatal(err)
				}
			}
		})
	}
}
