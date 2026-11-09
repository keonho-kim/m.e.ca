// Purpose: Measure repeated parsing of an unchanged large prepared-call argument body.
// Pattern: Tool boundary allocation benchmark.
// Usage: go test ./internal/tools/local -run ^$ -bench BenchmarkRepeatedArguments -benchmem.
// Related: internal/tools/local/catalog/arguments.go, internal/tools/local/prepare.go.
package local

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func BenchmarkRepeatedArguments(b *testing.B) {
	body, _ := json.Marshal(map[string]string{"path": "fixture.go", "content": strings.Repeat("source line\n", 40000)})
	call := toolapi.Call{ID: "call_fixture", Name: "write_file", Arguments: body}
	e := &Executor{}
	if _, _, err := e.parsedCall(call); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, _, err := e.parsedCall(call); err != nil {
			b.Fatal(err)
		}
	}
}
