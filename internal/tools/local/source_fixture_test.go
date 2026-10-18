// Purpose: Obtain exact fixture source for file-tool tests.
// Pattern: Test fixture.
// Usage: File mutation tests call observedEdit with inclusive line numbers.
// Related: internal/tools/local/files_test.go, internal/tools/local/approval_candidate_test.go, internal/tools/local/textedit/edit.go.
package local

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"testing"
)

func observedEdit(t *testing.T, e *Executor, path string, first, last int, replacement string) toolapi.Call {
	t.Helper()
	target, err := e.resolve(policy.Directory, path)
	if err != nil {
		t.Fatal(err)
	}
	body, err := target.root.ReadFile(target.relative)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(body), "\n")
	if first < 1 || last > len(lines) {
		t.Fatal("missing source range")
	}
	old := strings.Join(lines[first-1:last], "")
	args, err := json.Marshal(map[string]any{"path": path, "edits": []map[string]string{{"old_text": old, "new_text": replacement}}})
	if err != nil {
		t.Fatal(err)
	}
	return call("edit_file", string(args))
}
