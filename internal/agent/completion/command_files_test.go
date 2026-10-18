// Purpose: Preserve many observed shell effects as one historical command within the aggregate budget.
// Pattern: State contract regression test.
// Usage: go test ./internal/agent/completion -run TestCommandManyFiles.
// Related: internal/agent/completion/state.go.
package completion

import (
	"fmt"
	"testing"
)

func TestCommandManyFiles(t *testing.T) {
	r := Receipt{ID: "shell", Tool: "execute_command", Kind: "command", Command: "rm -rf generated", Success: true, Changed: true, Versions: map[string]string{}}
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/project/generated/%d", i)
		r.Targets = append(r.Targets, path)
		r.Versions[path] = "missing"
	}
	var state State
	state.Observe(r, r.Targets, false)
	if err := state.Check(); err != nil {
		t.Fatal(err)
	}
	if len(state.Receipts) != 1 || len(state.Facts().Commands) != 1 || len(state.Facts().Files) != 100 {
		t.Fatal("command duplicated or file effects lost")
	}
}
