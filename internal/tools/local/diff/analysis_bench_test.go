// Purpose: Measure complete preview and exact change-count preparation together.
// Pattern: File change preparation benchmark.
// Usage: go test ./internal/tools/local/diff -run '^$' -bench BenchmarkAnalysis -benchmem.
// Related: internal/tools/local/diff/diff.go, internal/tools/local/diff/count.go.
package diff

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkAnalysis(b *testing.B) {
	var source strings.Builder
	for i := 0; i < 8192; i++ {
		fmt.Fprintf(&source, "line %d: unchanged source content\n", i)
	}
	before := []byte(source.String())
	after := []byte(strings.Replace(source.String(), "line 4096:", "changed 4096:", 1))
	for _, test := range []struct {
		name          string
		before, after []byte
	}{
		{"SmallEdit", before, after},
		{"Create", nil, after},
		{"Replace", []byte(strings.Repeat("old\n", 300)), []byte(strings.Repeat("new\n", 300))},
	} {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				preview, added, deleted, err := Analyze("file", test.before, test.after, test.before != nil, true)
				if err != nil || preview == "" || added+deleted == 0 {
					b.Fatal("invalid analysis", err)
				}
			}
		})
	}
}
