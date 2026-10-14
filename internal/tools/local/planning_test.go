// Purpose: Verify Plan mode permits inspection but blocks every mutation and shell operation under auto approval.
// Pattern: Tool boundary contract test.
// Usage: go test ./internal/tools/local -run TestPlanReadOnly.
// Related: internal/tools/local/prepare.go, internal/tools/local/catalog.go, internal/policy/policy.go.
package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/policy"
)

func TestPlanReadOnlyWithAutomaticApproval(t *testing.T) {
	executor, project := testExecutor(t)
	if err := os.WriteFile(filepath.Join(project, "existing"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Permission: policy.AllowAll, Scope: policy.System}
	for _, tc := range []struct{ name, args string }{
		{"write_file", `{"path":"new","content":"change"}`},
		{"edit_file", `{"path":"existing","start_anchor":"1#12345678","end_anchor":"1#12345678","new_text":"change"}`},
		{"delete_file", `{"path":"existing"}`},
		{"move_file", `{"path":"existing","destination":"moved"}`},
		{"create_directory", `{"path":"new-dir"}`},
		{"execute_command", `{"command":"printf unsafe > new"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if op, err := executor.Prepare(context.Background(), settings, true, call(tc.name, tc.args)); err == nil || op != nil || !strings.Contains(err.Error(), "Plan approval is required") {
				t.Fatalf("Plan policy did not block the side effect: %v", err)
			}
		})
	}
	op, err := executor.Prepare(context.Background(), settings, true, call("read_file", `{"path":"existing"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(project, "existing"))
	if err != nil || string(content) != "original" {
		t.Fatalf("source changed: %q %v", content, err)
	}
}
