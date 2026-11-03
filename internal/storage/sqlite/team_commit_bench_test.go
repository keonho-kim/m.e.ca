// Purpose: Measure root journal commits for an execution reference and its public fact.
// Pattern: Comparable split and grouped durability benchmark.
// Usage: Compare transaction and allocation overhead on temporary session databases.
// Related: internal/storage/sqlite/team_messages.go, internal/storage/sqlite/commit.go.
package sqlite

import (
	"context"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func BenchmarkTeamFactCommit(b *testing.B) {
	for _, grouped := range []bool{false, true} {
		name := "split"
		if grouped {
			name = "grouped"
		}
		b.Run(name, func(b *testing.B) {
			root := benchmarkPayloadStore(b)
			ctx := context.Background()
			ref := &runrecord.EventReference{Session: "session_original", Sequence: 1}
			event := runrecord.Event{Kind: "team_reference", Agent: "Hana", Reference: ref}
			message := channel.Message{From: "Hana", Text: "write_file changed api.go", Observation: true}
			if grouped {
				message.Reference = ref
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if !grouped {
					if err := root.RecordEvent(ctx, root.session.ID, event); err != nil {
						b.Fatal(err)
					}
				}
				if _, err := root.AppendTeamMessage(ctx, message); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
