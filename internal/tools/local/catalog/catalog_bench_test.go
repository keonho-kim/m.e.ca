// Purpose: Measure fixed-catalog allocation costs at the tool argument and definition boundaries.
// Pattern: Synthetic tool contract benchmark.
// Usage: go test ./internal/tools/local/catalog -run '^$' -bench BenchmarkCatalog -benchmem.
// Related: internal/tools/local/catalog/catalog.go, internal/tools/local/catalog/arguments.go.
package catalog

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"testing"
)

func BenchmarkCatalog(b *testing.B) {
	b.Run("arguments", func(b *testing.B) {
		invocation := toolapi.Call{ID: "call_catalog", Name: "edit_file", Arguments: json.RawMessage(`{"path":"a","edits":[{"old_text":"old","new_text":"new"}]}`)}
		b.ReportAllocs()
		for b.Loop() {
			if _, _, err := Parse(invocation); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("definitions", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if len(Definitions()) != 4 {
				b.Fatal("incomplete catalog")
			}
		}
	})
}
