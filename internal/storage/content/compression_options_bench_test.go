// Purpose: Compare compression settings on equal payloads including ratio and construction cost.
// Pattern: Compression selection measurement.
// Usage: go test ./internal/storage/content -run '^$' -bench BenchmarkCompressionOptions -benchmem.
// Related: internal/storage/content/compression.go, internal/storage/content/compression_bench_test.go.
package content

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zstd"
)

func BenchmarkCompressionOptions(b *testing.B) {
	var source strings.Builder
	for i := 0; source.Len() < 1024*1024; i++ {
		fmt.Fprintf(&source, "func Handle%d(request Request) Result {\n  value := request.Value * %d\n  return Result{Value: value, Status: \"completed\"}\n}\n", i, i%37)
	}
	raw, _ := json.Marshal(map[string]string{"content": source.String()})
	for _, size := range []int{64 * 1024, len(raw)} {
		raw := raw[:size]
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.Run("S2", func(b *testing.B) {
				var buffer []byte
				b.ReportAllocs()
				for b.Loop() {
					buffer = s2.EncodeBetter(buffer, raw)
					b.ReportMetric(float64(len(buffer)), "compressed-bytes")
				}
			})
			for _, window := range []int{32 * 1024, 64 * 1024, 128 * 1024, 256 * 1024} {
				for _, low := range []bool{false, true} {
					b.Run(fmt.Sprintf("Zstd-%dk-low-%t", window/1024, low), func(b *testing.B) {
						encoder, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), zstd.WithWindowSize(window), zstd.WithEncoderLevel(zstd.SpeedFastest), zstd.WithLowerEncoderMem(low))
						if err != nil {
							b.Fatal(err)
						}
						defer encoder.Close()
						var buffer []byte
						b.ReportAllocs()
						for b.Loop() {
							buffer = encoder.EncodeAll(raw, buffer[:0])
							b.ReportMetric(float64(len(buffer)), "compressed-bytes")
						}
					})
				}
			}
		})
	}
}
