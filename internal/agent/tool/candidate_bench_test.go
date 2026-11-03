// Purpose: Measure approval binding without allocating an argument-sized concatenation.
// Pattern: Trust-boundary allocation benchmark and digest compatibility check.
// Usage: go test ./internal/agent/tool -run TestCandidateDigest -bench BenchmarkCandidateBinding -benchmem.
// Related: internal/agent/tool/candidate.go, internal/agent/run/prepare_execution.go.
package tool

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestCandidateDigestBindsExactNativeBytes(t *testing.T) {
	call := Call{ID: "call", Name: "write_file", Arguments: []byte(`{"path":"a","content":"value"}`)}
	want := fmt.Sprintf("%x", sha256.Sum256(append([]byte(call.Name+"\n"), call.Arguments...)))
	if got := BindCandidate(call, nil).Binding; got != want {
		t.Fatalf("binding=%s expected=%s", got, want)
	}
}
func BenchmarkCandidateBinding(b *testing.B) {
	call := Call{ID: "call", Name: "write_file", Arguments: []byte(`{"path":"a","content":"` + strings.Repeat("x", 1<<20) + `"}`)}
	b.ReportAllocs()
	for b.Loop() {
		if len(BindCandidate(call, nil).Binding) != 64 {
			b.Fatal("invalid binding")
		}
	}
}
