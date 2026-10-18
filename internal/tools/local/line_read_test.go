// Purpose: Verify internal source ranges, partial lines and actual edit line positions.
// Pattern: Filesystem integration tests.
// Usage: go test ./internal/tools/local -run TestLineRead.
// Related: internal/tools/local/mutations.go.
package local

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"testing"
)

func TestLineReadAndEditReportsPositions(t *testing.T) {
	executor, root := testExecutor(t)
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("first\nsecond\nlast\n"), 0600); err != nil {
		t.Fatal(err)
	}
	call := observedEdit(t, executor, "a", 2, 2, "new\nextra\n")
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory, Permission: policy.Approve}, call)
	if err != nil {
		t.Fatal(err)
	}
	if !op.Approval().Required || op.Approval().Candidate.Lines() != 3 {
		t.Fatal("edit bypassed review or approval")
	}
	result, err := op.Execute(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Lines []struct{ OldStartLine, OldEndLine, NewStartLine, NewEndLine int }
	}
	if err := json.Unmarshal(result.Content, &output); err != nil {
		t.Fatal(err)
	}
	if output.Lines[0].OldStartLine != 2 || output.Lines[0].OldEndLine != 2 || output.Lines[0].NewStartLine != 2 || output.Lines[0].NewEndLine != 3 || result.Changes[0].Lines == nil {
		t.Fatalf("missing positions: %s", result.Content)
	}
	body, err := os.ReadFile(filepath.Join(root, "a"))
	if err != nil || string(body) != "first\nnew\nextra\nlast\n" {
		t.Fatalf("%q %v", body, err)
	}
}
func TestExactEditPreservesCRLF(t *testing.T) {
	executor, root := testExecutor(t)
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("first\r\nsecond\r\nlast"), 0600); err != nil {
		t.Fatal(err)
	}
	edit := observedEdit(t, executor, "a", 2, 2, "new\n")
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory, Permission: policy.AllowAll}, edit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(root, "a"))
	if err != nil || string(body) != "first\r\nnew\r\nlast" {
		t.Fatalf("line ending lost: %q %v", body, err)
	}
}
