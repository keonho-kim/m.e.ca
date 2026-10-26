// Purpose: Compare payload compression time, allocations and stored size on identical JSON.
// Pattern: Codec regression benchmark with a standard-library baseline.
// Usage: go test ./internal/storage/content -run '^$' -bench BenchmarkPayloadCodec -benchmem.
// Related: internal/storage/content/compression.go.
package content

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func BenchmarkPayloadCodec(b *testing.B) {
	for _, size := range []int{64 * 1024, 1024 * 1024} {
		var source strings.Builder
		for i := 0; source.Len() < size; i++ {
			fmt.Fprintf(&source, "func Handle%d(request Request) Result {\n  value := request.Value * %d\n  return Result{Value: value, Status: \"completed\"}\n}\n", i, i%37)
		}
		raw, _ := json.Marshal(map[string]string{"path": "server.go", "content": source.String()})
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.Run("gzip", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var buffer bytes.Buffer
					writer, _ := gzip.NewWriterLevel(&buffer, gzip.BestSpeed)
					if _, err := writer.Write(raw); err != nil {
						b.Fatal(err)
					}
					if err := writer.Close(); err != nil {
						b.Fatal(err)
					}
					b.ReportMetric(float64(buffer.Len()), "compressed-bytes")
				}
			})
			b.Run("zstd", func(b *testing.B) {
				var codec Codec
				defer codec.Close()
				b.ReportAllocs()
				for b.Loop() {
					body, err := codec.Encode(raw)
					if err != nil {
						b.Fatal(err)
					}
					b.ReportMetric(float64(len(body)), "compressed-bytes")
				}
			})
			var buffer bytes.Buffer
			writer, _ := gzip.NewWriterLevel(&buffer, gzip.BestSpeed)
			writer.Write(raw)
			writer.Close()
			b.Run("gzip_decode", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					reader, err := gzip.NewReader(bytes.NewReader(buffer.Bytes()))
					if err != nil {
						b.Fatal(err)
					}
					if _, err := io.ReadAll(reader); err != nil {
						b.Fatal(err)
					}
					reader.Close()
				}
			})
			b.Run("zstd_decode", func(b *testing.B) {
				var codec Codec
				defer codec.Close()
				body, err := codec.Encode(raw)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				for b.Loop() {
					if _, err := codec.decode(body, len(raw)); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
