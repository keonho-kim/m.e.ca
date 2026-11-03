// Purpose: Measure eight receivers with repository-only and shared public-body admission.
// Pattern: Fixed committed-message workload benchmark.
// Usage: go test ./internal/agent/team -run '^$' -bench BenchmarkPublicHistoryFanout -benchmem.
// Related: internal/agent/team/public_history.go, internal/storage/sqlite/team_messages.go.
package team

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

func BenchmarkPublicHistoryFanout(b *testing.B) {
	for _, size := range []int{4096, 65536, 1048576} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			root := b.TempDir()
			catalog, err := sqlite.OpenSessions(filepath.Join(root, "sessions"))
			if err != nil {
				b.Fatal(err)
			}
			store, err := catalog.Create(context.Background(), root, root)
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { store.Close() })
			message, err := store.AppendTeamMessage(context.Background(), channel.Message{From: "Hana", Text: strings.Repeat("x", size)})
			if err != nil {
				b.Fatal(err)
			}
			for _, shared := range []bool{false, true} {
				b.Run(fmt.Sprintf("shared%v", shared), func(b *testing.B) {
					var history publicHistory
					history.append(message, message.Sequence-1)
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						for range 8 {
							if shared {
								values, err := history.read(context.Background(), store, message.Sequence-1, message.Sequence, 100)
								if err != nil || len(values) != 1 {
									b.Fatal(err)
								}
							} else {
								values, err := store.TeamMessages(context.Background(), message.Sequence-1, message.Sequence, 100)
								if err != nil || len(values) != 1 || values[0].Context().Content == "" {
									b.Fatal(err)
								}
							}
						}
					}
				})
			}
		})
	}
}
