// Purpose: Preserve longest-match and Unicode masking while measuring many shared key prefixes.
// Pattern: Secret boundary regression tests and workload benchmark.
// Usage: go test ./internal/config/credentials -run TestIndexedStream -bench BenchmarkStreamMaskKeys.
// Related: internal/redaction/matcher.go, internal/redaction/stream.go.
package redaction

import (
	"fmt"
	"strings"
	"testing"
)

func TestIndexedStreamSharedPrefixes(t *testing.T) {
	v := &Registry{}
	for _, key := range []string{"sk-common", "sk-common-123", "sk-common-456", "éclair", "élève", "한글키", "한글키-value"} {
		if err := v.Remember(key); err != nil {
			t.Fatal(err)
		}
	}
	for _, text := range []string{"sk-common-123!", "s\x00k-common-456!", "écl\x1bair!", "élève!", "한\x00글키-value!", "prefix 한글키!", "ordinary text"} {
		for step := 1; step <= len(text); step++ {
			mask := v.NewStreamMask()
			for at := 1; at <= len(text); at += step {
				mask(text[:at], false)
			}
			if got, want := mask(text, true), v.Redact(text); got != want {
				t.Fatalf("step=%d got=%q want=%q", step, got, want)
			}
		}
	}
}

func BenchmarkStreamMaskKeys(b *testing.B) {
	text := strings.Repeat("ordinary response text.\n", 45000)
	for _, count := range []int{1, 32, 256, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			v := &Registry{}
			for i := range count {
				if err := v.Remember(fmt.Sprintf("sk-audit-%04d-%032d", i, i)); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(text)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if len(v.NewStreamMask()(text, true)) != len(text) {
					b.Fatal("unexpected masking")
				}
			}
		})
	}
}
