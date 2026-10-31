// Purpose: Verify native source caching preserves provider bytes and detects mutable call changes.
// Pattern: Differential request and resource regression tests.
// Usage: go test ./internal/provider/protocol -run TestSourceHistory.
// Related: internal/provider/protocol/source_history.go, internal/provider/protocol/encoder_test.go.
package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestSourceHistoryPreservesNativeRequests(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.VLLM, connection.LMStudio, connection.Ollama, connection.SGLang, connection.LiteLLM} {
		t.Run(string(kind), func(t *testing.T) {
			var encoder Encoder
			defer encoder.Clear()
			source := []modelapi.Message{{Role: "assistant", Content: "Writing", Calls: []toolapi.Call{{ID: "one", Name: "write_file", Arguments: json.RawMessage(`{"path":"a","content":"` + strings.Repeat("x", 1<<20) + `"}`)}}}, {Role: "tool", ToolCallID: "one", ToolName: "write_file", Content: `{"ok":true}`}}
			request := modelapi.Request{ModelID: "model", MaxOutput: 123}
			for step := range 3 {
				if step == 2 {
					source[0].Calls[0].Arguments[9] = 'b'
				}
				plain, err := Messages(kind, source)
				if err != nil {
					t.Fatal(err)
				}
				_, want, err := Encode(kind, "https://model.test", nil, request, plain, Controls{})
				if err != nil {
					t.Fatal(err)
				}
				prepared, err := encoder.Messages(kind, request.ModelID, source)
				if err != nil {
					t.Fatal(err)
				}
				if step == 1 && prepared[0].encoded == nil {
					t.Fatal("unchanged native history was converted again")
				}
				_, body, err := encoder.Encode(kind, "https://model.test", nil, request, prepared, Controls{})
				if err != nil {
					t.Fatal(err)
				}
				got, err := io.ReadAll(body.Reader())
				if err != nil {
					t.Fatal(err)
				}
				var a, b any
				if json.Unmarshal(want, &a) != nil || json.Unmarshal(got, &b) != nil {
					t.Fatal("invalid request")
				}
				first, _ := json.Marshal(a)
				second, _ := json.Marshal(b)
				if string(first) != string(second) {
					t.Fatal("cached source request changed provider semantics")
				}
			}
			source[0].Calls[0].Arguments[0] = '!'
			if _, err := encoder.Messages(kind, request.ModelID, source); err == nil || encoder.history != nil {
				t.Fatal("invalid mutated arguments were trusted")
			}
		})
	}
}

func BenchmarkNativeHistoryEncoding(b *testing.B) {
	for _, size := range []int{1 << 20, 8 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			source := []modelapi.Message{{Role: "assistant", Calls: []toolapi.Call{{ID: "one", Name: "write_file", Arguments: json.RawMessage(`{"path":"a","content":"` + strings.Repeat("x", size) + `"}`)}}}}
			request := modelapi.Request{ModelID: "model", MaxOutput: 123}
			var encoder Encoder
			defer encoder.Clear()
			for i := 0; i < 2; i++ {
				messages, err := encoder.Messages(connection.OpenRouter, request.ModelID, source)
				if err != nil {
					b.Fatal(err)
				}
				if _, _, err = encoder.Encode(connection.OpenRouter, "https://model.test", nil, request, messages, Controls{}); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				messages, err := encoder.Messages(connection.OpenRouter, request.ModelID, source)
				if err != nil {
					b.Fatal(err)
				}
				if _, _, err = encoder.Encode(connection.OpenRouter, "https://model.test", nil, request, messages, Controls{}); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(encoder.retained), "retained-B")
		})
	}
}
