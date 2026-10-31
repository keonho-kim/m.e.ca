// Purpose: Measure context fallback accounting without allocating conversation-sized JSON.
// Pattern: Large-history runtime benchmark.
// Usage: go test ./internal/agent/run -run '^$' -bench BenchmarkUsageFallback -benchmem.
// Related: internal/agent/run/request_usage.go.
package run

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"strings"
	"testing"
)

func BenchmarkUsageFallback(b *testing.B) {
	store := &usageRecords{}
	s := execution{runner: Runner{Memory: &compact.Controller{Repository: store, Session: "s"}}}
	request := modelapi.Request{Messages: []modelapi.Message{{Role: "user", Content: strings.Repeat("x", 8<<20)}}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.counts = store.counts[:0]
		if err := s.countRequest(context.Background(), request, nil, 0, 0); err != nil {
			b.Fatal(err)
		}
	}
}
