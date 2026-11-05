// Purpose: Verify reused shell syntax and environment remain immutable across analyses.
// Pattern: Concurrent equivalence test against independent parsing.
// Usage: go test -race ./internal/policy/command -run TestProgramReuse.
// Related: internal/policy/command/program.go, internal/policy/command/bindings.go.
package command

import (
	"reflect"
	"sync"
	"testing"
)

func TestProgramReusePreservesBranchAnalysis(t *testing.T) {
	input := testContext()
	text := `V=/outside; f() { rm "$V/$1"; }; (V=/project; f file); f other; printf '{a,b}'`
	want, err := Analyze(t.Context(), text, input)
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse(text, input.Environment)
	if err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 10 {
				got, err := program.Analyze(t.Context(), input.Context)
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Errorf("reused syntax changed safety: %+v %v", got, err)
					return
				}
			}
		})
	}
	workers.Wait()
}
