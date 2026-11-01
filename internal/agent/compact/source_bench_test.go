// Purpose: Measure ordered compaction preprocessing independently of model latency.
// Pattern: Bounded pure-transformation benchmark.
// Usage: Run BenchmarkSourceSelection with -benchmem.
// Related: internal/agent/compact/source.go, internal/agent/compact/controller.go.
package compact

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkSourceSelection(b *testing.B) {
	for _, duplicate := range []bool{false, true} {
		b.Run(fmt.Sprintf("duplicate%t", duplicate), func(b *testing.B) {
			records := make([]Record, 128)
			text := strings.Repeat("Observed source line.\n", 1024)
			for i := range records {
				records[i] = Record{Role: "outcome", Text: text}
				if !duplicate {
					records[i].Text += fmt.Sprint(i)
				}
			}
			b.ReportAllocs()
			for b.Loop() {
				sources := sourceBuffer{limit: 4 << 20}
				for _, record := range records {
					if err := sources.add(record, func(s string) string { return s }); err != nil {
						b.Fatal(err)
					}
				}
				_ = sources.packet()
			}
		})
	}
}
