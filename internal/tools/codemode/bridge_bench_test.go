// Purpose: Measure JSON transfer and rejected return allocations through the embedded VM.
// Pattern: Fixed host-result workloads without model or filesystem latency.
// Usage: go test ./internal/tools/codemode -run '^$' -bench 'CodeBridge|OversizedReturn' -benchmem.
// Related: internal/tools/codemode/engine.go, internal/tools/codemode/runtime.go.
package codemode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func BenchmarkCodeBridge(b *testing.B) {
	for _, size := range []int{64 << 10, 1 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			body, _ := json.Marshal(map[string]string{"text": strings.Repeat("x", size)})
			code := `const values = await Promise.all(Array.from({length:16}, () => tools.read({}))); return values.map(x => x.text.length);`
			dispatch := func(_ context.Context, calls []Call, deliver Deliver) error {
				for _, call := range calls {
					if _, err := deliver(call.ID, Reply{Value: body}); err != nil {
						return err
					}
				}
				return nil
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Run(b.Context(), code, []string{"read"}, dispatch); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkOversizedReturn(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Run(b.Context(), `return "x".repeat(8*1024*1024);`, nil, nil); err == nil {
			b.Fatal("oversized result accepted")
		}
	}
}
