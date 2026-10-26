// Purpose: Compare queued snapshots with latest-value delivery under a slow terminal consumer.
// Pattern: Synthetic stream delivery benchmark.
// Usage: go test ./internal/tui/session -run '^$' -bench BenchmarkStreamDelivery -benchmem.
// Related: internal/tui/session/delivery.go, internal/tui/session/tasks.go.
package session

import (
	"context"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func BenchmarkStreamDelivery(b *testing.B) {
	for _, latest := range []bool{false, true} {
		name := "queued"
		if latest {
			name = "latest"
		}
		b.Run(name, func(b *testing.B) {
			queue := make(chan notice, 64)
			body := strings.Repeat("x", 64*1024)
			delivery := runtimeDelivery{}
			b.ReportAllocs()
			for b.Loop() {
				for i := range 32 {
					event := runrecord.Event{Kind: "stream", Text: body[:(i+1)*2048]}
					if latest {
						if err := delivery.stream(context.Background(), queue, event); err != nil {
							b.Fatal(err)
						}
					} else {
						copy := event
						queue <- notice{Event: &copy}
					}
				}
				for len(queue) > 0 {
					n := <-queue
					if n.Stream != nil {
						n.Stream.take()
					}
				}
			}
		})
	}
}
