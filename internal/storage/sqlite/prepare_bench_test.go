// Purpose: Measure the write-lock interval with identical record preparation inside or outside SQLite.
// Pattern: Fixed durability-boundary benchmark with separate transaction timing.
// Usage: go test ./internal/storage/sqlite -run '^$' -bench BenchmarkJournalTransaction -benchmem.
// Related: internal/storage/sqlite/prepare.go, internal/storage/sqlite/commit.go.
package sqlite

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

func BenchmarkJournalTransaction(b *testing.B) {
	for _, outside := range []bool{false, true} {
		b.Run(fmt.Sprintf("prepare_outside=%v", outside), func(b *testing.B) {
			s := benchmarkPayloadStore(b)
			var held time.Duration
			step := 0
			b.ReportAllocs()
			for b.Loop() {
				messages := make([]model.Message, 4)
				entries := make([]record.Entry, len(messages))
				for index := range messages {
					messages[index] = model.Message{Role: "assistant", Content: fmt.Sprintf("%d:%d:", step, index) + strings.Repeat("x", 128<<10)}
					entries[index].Message = &messages[index]
				}
				step++
				if err := checkJournal(b.Context(), entries); err != nil {
					b.Fatal(err)
				}
				var prepared []preparedRecord
				if outside {
					var err error
					prepared, err = s.prepareJournal(b.Context(), entries)
					if err != nil {
						b.Fatal(err)
					}
				}
				tx, err := s.db.BeginTx(b.Context(), nil)
				if err != nil {
					b.Fatal(err)
				}
				start := time.Now()
				encoder := journalEncoder{store: s, ctx: b.Context(), lookup: tx}
				for index, entry := range entries {
					var value preparedRecord
					if outside {
						value = prepared[index]
					} else {
						value, err = encoder.prepare(entry)
					}
					if err == nil {
						err = value.write(b.Context(), s, tx, s.Session().ID, "now", &journalReceipt{})
					}
					if err != nil {
						_ = tx.Rollback()
						b.Fatal(err)
					}
				}
				if err := tx.Commit(); err != nil {
					b.Fatal(err)
				}
				held += time.Since(start)
			}
			b.ReportMetric(float64(held.Nanoseconds())/float64(b.N), "tx-ns/op")
		})
	}
}
