// Purpose: Measure prepared immutable history independently of defensive mutable-input hashing.
// Pattern: Native request workload benchmark.
// Usage: go test ./internal/provider/protocol -bench BenchmarkOwnedNativeHistory -benchmem.
// Related: internal/agent/model/native_identity.go, internal/provider/protocol/source_history.go.
package protocol

import (
	"fmt"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func BenchmarkOwnedNativeHistory(b *testing.B) {
	for _, size := range []int{1 << 20, 8 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			m := modelapi.Message{Role: "assistant", Calls: []toolapi.Call{{ID: "one", Name: "write_file", Arguments: []byte(`{"path":"a","content":"` + strings.Repeat("x", size) + `"}`)}}}
			source := []modelapi.Message{m.SealHistory()}
			var encoder Encoder
			defer encoder.Clear()
			request := modelapi.Request{ModelID: "model", MaxOutput: 32}
			encode := func() {
				messages, err := encoder.Messages(connection.OpenRouter, "model", source)
				if err != nil {
					b.Fatal(err)
				}
				if _, _, err = encoder.Encode(connection.OpenRouter, "https://model", nil, request, messages, Controls{}); err != nil {
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
			b.ReportMetric(float64(encoder.retained), "retained-B")
		})
	}
}
