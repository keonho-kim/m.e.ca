// Purpose: Permit ordinary shell reads while routing explicit output-file mutations to file tools.
// Pattern: Shell policy regression tests.
// Usage: go test ./internal/policy/shell -run TestShellReads.
// Related: internal/policy/shell/check.go.
package shell

import "testing"

func TestShellReadsAndOutputMutations(t *testing.T) {
	for _, command := range []string{"cat pyproject.toml", "uv tree && cat .python-version", "head -20 README.md", "tail -n 10 log", "sed -n '1,80p' app.py", "nl -ba app.py | sed -n '10,30p'", "cat absent 2>/dev/null"} {
		if err := Check(command); err != nil {
			t.Errorf("read rejected: %s: %v", command, err)
		}
	}
	for _, command := range []string{"cat > source.py <<'EOF'\nprint(1)\nEOF", "printf code > source.py", "cat source.py | tee other.py", "cat source.py >> other.py"} {
		if err := Check(command); err == nil {
			t.Errorf("direct mutation accepted: %s", command)
		}
	}
}
