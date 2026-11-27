// Purpose: Measure repeated native request encoding with the product's full tool catalog.
// Pattern: Warm serial encoder workload over fixed schemas and changing output limits.
// Usage: Run BenchmarkToolSchemaEncoding with -benchmem.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/tool_snapshot.go.
package protocol

import (
	"io"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tools/codemode"
	"github.com/keonho-kim/meca/internal/tools/local/catalog"
)

func BenchmarkToolSchemaEncoding(b *testing.B) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.LMStudio, connection.Ollama, connection.Claude, connection.Gemini} {
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

func BenchmarkCodeModeRequest(b *testing.B) {
	definitions := catalog.Definitions()
	for _, code := range []bool{false, true} {
		name := "native"
		if code {
			name = "codemode"
		}
		b.Run(name, func(b *testing.B) {
			var encoder Encoder
			defer encoder.Clear()
			request := modelapi.Request{ModelID: "model", MaxOutput: 8192}
			messages := []Message{{Role: "user", Content: "Inspect the project and report the results."}}
			requestBytes := 0
			request.Tools = definitions
			if code {
				request.Tools = []toolapi.Definition{codemode.Definition(definitions)}
			}
			b.ReportAllocs()
			for b.Loop() {
				_, body, err := encoder.Encode(connection.VLLM, "https://model.test", nil, request, messages, Controls{})
				if err != nil {
					b.Fatal(err)
				}
				encoded, err := io.ReadAll(body.Reader())
				if err != nil {
					b.Fatal(err)
				}
				requestBytes = len(encoded)
			}
			b.ReportMetric(float64(requestBytes), "request-B")
		})
	}
}
