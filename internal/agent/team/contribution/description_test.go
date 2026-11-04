// Purpose: Measure retained descriptions independently of accepted source size.
// Pattern: Focused allocation benchmark.
// Usage: go test ./internal/agent/team/contribution -run '^$' -bench BenchmarkDescribe -benchmem.
// Related: internal/agent/team/contribution/description.go, internal/agent/team/contribution/state.go.
package contribution

import (
	"crypto/sha256"
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

func TestDescriptionFingerprintsCoverAllAcceptedText(t *testing.T) {
	for _, size := range []int{0, 4096, 4097, 16384, 20000, 1 << 20} {
		text := strings.Repeat("a", size)
		value := Describe(Scope{Summary: text, Contract: text + "tail"})
		if value.summaryHash != sha256.Sum256([]byte(text)) || value.contractHash != sha256.Sum256([]byte(text+"tail")) {
			t.Fatalf("incorrect fingerprint at %d bytes", size)
		}
		if len(value.Summary) > 4096 || len(value.Contract) > 4096 {
			t.Fatal("retained input was not bounded")
		}
	}
}
