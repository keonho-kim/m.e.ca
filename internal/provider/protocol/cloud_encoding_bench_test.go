// Purpose: Compare repeated native tool-history conversion and encoding with identical requests.
// Pattern: Execution-scoped warm/cold encoding benchmark.
// Usage: go test ./internal/provider/protocol -run '^$' -bench BenchmarkCloudConversationEncoding -benchmem.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/claude_bench_test.go.
package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func BenchmarkCloudConversationEncoding(b *testing.B) {
	arguments := json.RawMessage(`{"content":"` + strings.Repeat("observed text ", 2048) + `"}`)
	var source []modelapi.Message
	for i := range 32 {
		id := fmt.Sprintf("call_%d", i)
		source = append(source, modelapi.Message{Role: "assistant", Calls: []toolapi.Call{{ID: id, Name: "write_file", Arguments: arguments}}}, modelapi.Message{Role: "tool", ToolCallID: id, ToolName: "write_file", Content: "done"})
	}
	for _, kind := range []connection.ProviderKind{connection.Claude, connection.Gemini} {
		for _, cached := range []bool{false, true} {
			b.Run(fmt.Sprintf("%s/cache=%v", kind, cached), func(b *testing.B) {
				var encoder *Encoder
				if cached {
					encoder = &Encoder{}
					defer encoder.Clear()
				}
				request := modelapi.Request{ModelID: "model", MaxOutput: 32}
				messages, err := encoder.Messages(kind, request.ModelID, source)
				if err != nil {
					b.Fatal(err)
				}
				if _, _, err := encoder.Encode(kind, "https://model.test", nil, request, messages, Controls{}); err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				for b.Loop() {
					messages, err := encoder.Messages(kind, request.ModelID, source)
					if err != nil {
						b.Fatal(err)
					}
					_, body, err := encoder.Encode(kind, "https://model.test", nil, request, messages, Controls{})
					if err != nil || body.Len() == 0 {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
