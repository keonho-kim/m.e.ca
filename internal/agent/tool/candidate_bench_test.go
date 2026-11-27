// Purpose: Measure approval binding without allocating an argument-sized concatenation.
// Pattern: Trust-boundary allocation benchmark.
// Usage: go test ./internal/agent/tool -run '^$' -bench BenchmarkCandidateBinding -benchmem.
// Related: internal/agent/tool/candidate.go, internal/agent/run/prepare_execution.go.
package tool

import (
	"strings"
	"testing"
)

func BenchmarkCandidateBinding(b *testing.B) {
	call := Call{ID: "call", Name: "write_file", Arguments: []byte(`{"path":"a","content":"` + strings.Repeat("x", 1<<20) + `"}`)}
	b.ReportAllocs()
	for b.Loop() {
		if BindCandidate(call, nil).Binding == "" {
			b.Fatal("invalid binding")
		}
	}
}
