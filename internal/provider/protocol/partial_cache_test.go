// Purpose: Preserve bounded prefix reuse and exact provider semantics across oversized history.
// Pattern: Differential protocol and resource regression tests.
// Usage: go test ./internal/provider/protocol -run TestPartial -bench BenchmarkPartialHistory.
// Related: internal/provider/protocol/encoded_history.go, internal/provider/protocol/source_history.go.
package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestPartialHistoryKeepsPrefixAcrossOversizedNativeSuffix(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.VLLM, connection.SGLang, connection.LiteLLM, connection.LMStudio, connection.Ollama} {
		t.Run(string(kind), func(t *testing.T) {
			e := NewEncoder(128 * 1024)
			defer e.Clear()
			source := []modelapi.Message{{Role: "system", Content: strings.Repeat("instructions ", 6000)}, {Role: "assistant", Calls: []toolapi.Call{{ID: "a", Name: "write_file", Arguments: []byte(`{"path":"a","content":"` + strings.Repeat("x", 256*1024) + `"}`)}}}, {Role: "tool", ToolCallID: "a", ToolName: "write_file", Content: `{"changed":true}`}}
			request := modelapi.Request{ModelID: "model", MaxOutput: 32}
			var prefix []byte
			var first Body
			for step := range 3 {
				if step == 2 {
					source[1].Calls[0].Arguments[9] = 'b'
				}
				plain, err := Messages(kind, source)
				if err != nil {
					t.Fatal(err)
				}
				_, want, err := Encode(kind, "https://model", nil, request, plain, Controls{})
				if err != nil {
					t.Fatal(err)
				}
				prepared, err := e.Messages(kind, "model", source)
				if err != nil {
					t.Fatal(err)
				}
				if step > 0 && prepared[0].encoded == nil {
					t.Fatal("cached prefix was reconverted")
				}
				_, body, err := e.Encode(kind, "https://model", nil, request, prepared, Controls{})
				if err != nil {
					t.Fatal(err)
				}
				got, err := io.ReadAll(body.Reader())
				var a, b any
				if err != nil || json.Unmarshal(want, &a) != nil || json.Unmarshal(got, &b) != nil || !reflect.DeepEqual(a, b) {
					t.Fatal("overflow request changed", err)
				}
				if e.retained+e.tools.bytes > e.limit() || len(e.history) != 1 || len(e.groups) != 1 {
					t.Fatal("overflow retained a partial group or exceeded its budget")
				}
				if step > 0 && &prefix[0] != &e.history[0].json[0] {
					t.Fatal("prefix encoding replaced")
				}
				prefix = e.history[0].json
				if step == 0 {
					first = body
				}
			}
			e.Clear()
			if got, err := io.ReadAll(first.Reader()); err != nil || len(got) != first.Len() {
				t.Fatal("old overflow body lost replay ownership", err)
			}
		})
	}
}

func BenchmarkPartialHistory(b *testing.B) {
	for _, size := range []int{1 << 20, 2 << 20, 4 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			source := make([]modelapi.Message, 32)
			for i := range source {
				source[i] = modelapi.Message{Role: "user", Content: strings.Repeat("text ", size/32/5)}.SealHistory()
			}
			e := NewEncoder(2 << 20)
			defer e.Clear()
			request := modelapi.Request{ModelID: "model", MaxOutput: 128}
			encode := func() {
				messages, err := e.Messages(connection.OpenRouter, "model", source)
				if err != nil {
					b.Fatal(err)
				}
				_, body, err := e.Encode(connection.OpenRouter, "https://model", nil, request, messages, Controls{})
				if err != nil || body.Len() == 0 {
					b.Fatal(err)
				}
			}
			encode()
			encode()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				encode()
			}
			b.StopTimer()
			b.ReportMetric(float64(e.retained), "retained-B")
		})
	}
}
