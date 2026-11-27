// Purpose: Measure retained descriptions independently of accepted source size.
// Pattern: Focused allocation benchmark.
// Usage: go test ./internal/agent/team/contribution -run '^$' -bench BenchmarkDescribe -benchmem.
// Related: internal/agent/team/contribution/description.go, internal/agent/team/contribution/state.go.
package contribution

import (
	"strings"
	"testing"
)

func BenchmarkDescribe(b *testing.B) {
	for _, tc := range []struct {
		name string
		size int
	}{{"short", 64}, {"large", 1 << 20}} {
		b.Run(tc.name, func(b *testing.B) {
			source := Scope{Summary: strings.Repeat("s", tc.size), Contract: strings.Repeat("c", tc.size)}
			b.ReportAllocs()
			b.SetBytes(int64(2 * tc.size))
			b.ResetTimer()
			for b.Loop() {
				Describe(source)
			}
		})
	}
}
