// Purpose: Measure durable admission of a public context page into one private history.
// Pattern: Fixed message-group persistence benchmark.
// Usage: Run BenchmarkContextCommit with -benchtime=50x and -benchmem.
// Related: internal/storage/sqlite/commit.go, internal/storage/sqlite/messages.go.
package sqlite

import (
	"context"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func BenchmarkContextCommit(b *testing.B) {
	s := benchmarkPayloadStore(b)
	records := make([]runrecord.Entry, 64)
	for i := range records {
		records[i].Message = &modelapi.Message{Role: "user", Content: "Public work observation and current request."}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := s.CommitContext(context.Background(), s.Session().ID, records, int64(i+1)); err != nil {
			b.Fatal(err)
		}
	}
}
