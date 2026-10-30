// Purpose: Track text-only serialization and image data-URL allocation costs.
// Pattern: Protocol workload benchmarks.
// Usage: Run BenchmarkTextMessages and BenchmarkImageEncoding with -benchmem.
// Related: internal/provider/protocol/input.go, internal/provider/protocol/images.go.
package protocol

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func BenchmarkTextMessages(b *testing.B) {
	var messages []Message
	for range 40 {
		messages = append(messages, Message{Role: "user", Content: strings.Repeat("source code & 한국어 <module>\n", 128)})
	}
	payload := struct {
		Messages []Message `json:"messages"`
	}{messages}
	b.ReportAllocs()
	for b.Loop() {
		body, err := wireJSON.Marshal(payload)
		if err != nil {
			b.Fatal(err)
		}
		runtime.KeepAlive(body)
	}
}

func BenchmarkImageEncoding(b *testing.B) {
	for _, size := range []int{1 << 20, 8 << 20} {
		data := make([]byte, size)
		b.Run(fmt.Sprintf("%dMiB", size>>20), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				encoded, err := encodeImage(data, "data:image/png;base64,")
				if err != nil {
					b.Fatal(err)
				}
				runtime.KeepAlive(encoded)
			}
		})
	}
}
