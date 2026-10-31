// Purpose: Measure display-only argument decoding while large source fields are omitted.
// Pattern: Terminal event formatting benchmark.
// Usage: go test ./internal/tui/transcript -run '^$' -bench BenchmarkDisplayArguments -benchmem.
// Related: internal/tui/transcript/tool_arguments.go.
package transcript

import (
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func BenchmarkDisplayArguments(b *testing.B) {
	for _, size := range []int{4096, 1 << 20, 16 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			call := toolapi.Call{Name: "write_file", Arguments: json.RawMessage(`{"path":"a","content":"` + strings.Repeat("x", size) + `"}`)}
			f := Formatter{Redact: func(s string) string { return s }}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if f.arguments(call) == "" {
					b.Fatal("missing path")
				}
			}
		})
	}
}
