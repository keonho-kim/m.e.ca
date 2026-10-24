// Purpose: Measure fixed-catalog allocation costs at the tool argument and definition boundaries.
// Pattern: Synthetic tool contract benchmark.
// Usage: go test ./internal/tools/local -run '^$' -bench BenchmarkCatalog -benchmem.
// Related: internal/tools/local/catalog.go, internal/tools/local/arguments.go.
package local

import "testing"

func BenchmarkCatalog(b *testing.B) {
	b.Run("arguments", func(b *testing.B) {
		invocation := call("edit_file", `{"path":"a","edits":[{"old_text":"old","new_text":"new"}]}`)
		b.ReportAllocs()
		for b.Loop() {
			if _, _, err := parseArguments(invocation); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("definitions", func(b *testing.B) {
		e := &Executor{}
		b.ReportAllocs()
		for b.Loop() {
			if len(e.Definitions()) != 4 {
				b.Fatal("incomplete catalog")
			}
		}
	})
}
