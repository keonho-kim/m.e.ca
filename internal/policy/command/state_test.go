// Purpose: Verify shell branches isolate environment changes and measure compound-command analysis.
// Pattern: State-isolation regression tests and fixed-input benchmark.
// Usage: go test ./internal/policy/command -bench BenchmarkCompoundEnvironment -benchmem.
// Related: internal/policy/command/state.go, internal/policy/command/analyze.go.
package command

import (
	"fmt"
	"strings"
	"testing"
)

func TestEnvironmentBranchesKeepUnsetsAndUnknownsIsolated(t *testing.T) {
	for _, text := range []string{
		`X=/outside; (unset X); rm "$X/a"`,
		`X=/outside; env -u X true; rm "$X/a"`,
		`X=/outside; env -i true; rm "$X/a"`,
		`X=/outside; X=/project true; rm "$X/a"`,
		`X=/outside; if true; then X=/project; fi; rm "$X/a"`,
	} {
		report, err := Analyze(t.Context(), text, testContext())
		if err != nil || len(report.Safety.Reasons) == 0 {
			t.Fatalf("lost branch uncertainty for %s: %+v %v", text, report, err)
		}
	}
}

func BenchmarkCompoundEnvironment(b *testing.B) {
	input := testContext()
	for i := range 100 {
		input.Environment = append(input.Environment, fmt.Sprintf("VAR_%d=value%d", i, i))
	}
	text := "printf start" + strings.Repeat(" && printf next", 16) + " | tail -n 40"
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Analyze(b.Context(), text, input); err != nil {
			b.Fatal(err)
		}
	}
}
