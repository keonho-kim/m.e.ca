// Purpose: Measure durable session size for identical small records and compressed payload references.
// Pattern: Fixed-record-count storage benchmark.
// Usage: go test ./internal/storage/sqlite -run '^$' -bench BenchmarkSessionFootprint -benchtime=1x.
// Related: internal/storage/sqlite/schema.go, internal/storage/sqlite/prepare_bench_test.go.
package sqlite

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
)

func BenchmarkSessionFootprint(b *testing.B) {
	for _, size := range []int{128, 48 << 10} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			var total int64
			for b.Loop() {
				s := benchmarkPayloadStore(b)
				for group := range 10 {
					messages := make([]model.Message, 100)
					entries := make([]record.Entry, len(messages))
					for i := range messages {
						id := fmt.Sprintf("message_%d_%d", group, i)
						messages[i] = model.Message{ID: id, Role: "assistant", Content: id + strings.Repeat("x", size)}
						entries[i].Message = &messages[i]
					}
					if err := s.Commit(b.Context(), s.session.ID, entries); err != nil {
						b.Fatal(err)
					}
				}
				if err := s.Close(); err != nil {
					b.Fatal(err)
				}
				info, err := os.Stat(s.path)
				if err != nil {
					b.Fatal(err)
				}
				total += info.Size()
			}
			b.ReportMetric(float64(total)/float64(b.N), "db-bytes")
		})
	}
}
