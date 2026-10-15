// Purpose: Verify Plan queries cannot execute programs or mutate filesystem state.
// Pattern: Boundary parser tests.
// Usage: go test ./internal/policy/shell -run TestPlanQuery.
// Related: internal/policy/shell/query.go.
package shell

import "testing"

func TestPlanQuery(t *testing.T) {
	for _, text := range []string{"ls -la src", "find . -type f -name '*.go' -maxdepth 3", "rg --files -g '*.go' .", "rg -n -i 'TODO' src", "grep -rn 'TODO' src"} {
		if _, err := PlanQuery(text); err != nil {
			t.Errorf("%s: %v", text, err)
		}
	}
	for _, text := range []string{"find . -delete", "find . -exec touch x ;", "find . -fprint x", "rg --pre sh x", "ls > x", "ls | sh", "ls && touch x", "ls $(touch x)", "X=1 ls", "ls &", "sh -c ls", "./ls", "ls /tmp/*", "grep -f patterns ."} {
		if _, err := PlanQuery(text); err == nil {
			t.Errorf("unsafe or unsupported query accepted: %s", text)
		}
	}
}
