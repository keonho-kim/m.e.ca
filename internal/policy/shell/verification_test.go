// Purpose: Ensure verification commands propagate visible command failures.
// Pattern: Shell policy contract tests.
// Usage: go test ./internal/policy/shell -run TestVerification.
// Related: internal/policy/shell/verification.go.
package shell

import "testing"

func TestVerification(t *testing.T) {
	for _, text := range []string{"python3 -m unittest -v", "A=value python3 -m unittest", "test -s README.md && cmp README.md expected", "python3 -m unittest > result.log 2>&1"} {
		if err := CheckVerification(text); err != nil {
			t.Fatalf("%s: %v", text, err)
		}
	}
	for _, text := range []string{"false; echo ok", "false || true", "! false", "false | true", "false &", "if false; then true; fi", "sh -c 'false; true'", "echo $(false)", "exec false", "exit 0", "set -e; false", "f() { false; }; f"} {
		if err := CheckVerification(text); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
}
