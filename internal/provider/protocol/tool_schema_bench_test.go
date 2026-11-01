// Purpose: Measure repeated native request encoding with the product's full tool catalog.
// Pattern: Warm serial encoder workload over fixed schemas and changing output limits.
// Usage: Run BenchmarkToolSchemaEncoding with -benchmem.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/tool_snapshot.go.
package protocol

import (
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tools/local/catalog"
)

func BenchmarkToolSchemaEncoding(b *testing.B) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.LMStudio, connection.Ollama} {
		for _, size := range []int{100, 64 * 1024} {
			name := "small"
			if size > 100 {
				name = "large"
			}
			b.Run(string(kind)+"/"+name, func(b *testing.B) {
				var encoder Encoder
				defer encoder.Clear()
				request := modelapi.Request{ModelID: "model", MaxOutput: 8192, Tools: catalog.Definitions()}
				messages := []Message{{Role: "user", Content: strings.Repeat("x", size)}}
				for range 2 {
					if _, _, err := encoder.Encode(kind, "https://model.test", nil, request, messages, Controls{}); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportAllocs()
				for b.Loop() {
					request.MaxOutput++
					if _, _, err := encoder.Encode(kind, "https://model.test", nil, request, messages, Controls{}); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
