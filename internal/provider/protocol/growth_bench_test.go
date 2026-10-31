// Purpose: Measure full provider message conversion and request encoding with an unchanged history prefix.
// Pattern: Growing conversation workload benchmark.
// Usage: Run BenchmarkConversationEncoding with -benchmem before and after request caching changes.
// Related: internal/provider/protocol/request.go, internal/provider/protocol/tools.go.
package protocol

import (
	"fmt"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func BenchmarkConversationEncoding(b *testing.B) {
	for _, size := range []int{2048, 1 << 20, 8 << 20} {
		for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.LMStudio, connection.Ollama} {
			for _, cached := range []bool{false, true} {
				b.Run(fmt.Sprintf("%s/bytes%d/cache=%v", kind, size, cached), func(b *testing.B) {
					source := make([]modelapi.Message, 41)
					for i := range 40 {
						source[i] = modelapi.Message{Role: "user", Content: strings.Repeat("x", max(1, size/40))}
					}
					request := modelapi.Request{ModelID: "model", MaxOutput: 8192}
					var encoder *Encoder
					if cached {
						encoder = &Encoder{}
						defer encoder.Clear()
					}
					step := 0
					b.ReportAllocs()
					for b.Loop() {
						source[40] = modelapi.Message{Role: "user", Content: []string{"next 0", "next 1"}[step%2]}
						messages, err := Messages(kind, source)
						if err != nil {
							b.Fatal(err)
						}
						_, body, err := encoder.Encode(kind, "https://model.test", nil, request, messages, Controls{})
						if err != nil || body.Len() == 0 {
							b.Fatal(err)
						}
						step++
					}
				})
			}
		}
	}
}

func BenchmarkColdConversationEncoding(b *testing.B) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.LMStudio} {
		for _, cached := range []bool{false, true} {
			b.Run(fmt.Sprintf("%s/cache=%v", kind, cached), func(b *testing.B) {
				source := []modelapi.Message{{Role: "user", Content: strings.Repeat("x", 8<<20)}}
				request := modelapi.Request{ModelID: "model", MaxOutput: 8192}
				b.ReportAllocs()
				for b.Loop() {
					var encoder *Encoder
					if cached {
						encoder = &Encoder{}
					}
					messages, err := Messages(kind, source)
					if err != nil {
						b.Fatal(err)
					}
					_, body, err := encoder.Encode(kind, "https://model.test", nil, request, messages, Controls{})
					if err != nil || body.Len() == 0 {
						b.Fatal(err)
					}
					if encoder != nil {
						encoder.Clear()
					}
				}
			})
		}
	}
}
