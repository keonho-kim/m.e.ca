// Purpose: Verify visible line anchors, partial-line handling and changed-line reports through actual file tools.
// Pattern: Filesystem integration tests.
// Usage: go test ./internal/tools/local -run TestLineRead.
// Related: internal/tools/local/file/read.go, internal/tools/local/file/lines.go, internal/tools/local/mutations.go.
package local

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/config/credentials"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLineReadAndEditReportsPositions(t *testing.T) {
	executor, root := testExecutor(t)
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("first\nsecond\nlast\n"), 0600); err != nil {
		t.Fatal(err)
	}
	call := anchoredEdit(t, executor, "a", 2, 2, "new\nextra\n")
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
		Lines struct{ OldStartLine, OldEndLine, NewStartLine, NewEndLine int }
	}
	if err := json.Unmarshal(result.Content, &output); err != nil {
		t.Fatal(err)
	}
	if output.Lines.OldStartLine != 2 || output.Lines.OldEndLine != 2 || output.Lines.NewStartLine != 2 || output.Lines.NewEndLine != 3 || result.Changes[0].Lines == nil {
		t.Fatalf("missing positions: %s", result.Content)
	}
	body, err := os.ReadFile(filepath.Join(root, "a"))
	if err != nil || string(body) != "first\nnew\nextra\nlast\n" {
		t.Fatalf("%q %v", body, err)
	}
}
func TestLineReadPartialAndCRLF(t *testing.T) {
	executor, root := testExecutor(t)
	vault := &credentials.Vault{}
	executor.redact = vault.Redact
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("first\r\nsecond\r\nlast"), 0600); err != nil {
		t.Fatal(err)
	}
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory, Permission: policy.Approve}, call("read_file", `{"path":"a","offset":2,"limit":16}`))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	var read struct {
		Content                        string
		StartLine, EndLine, NextOffset int
		StartPartial, EndPartial       bool
	}
	if err := json.Unmarshal(result.Content, &read); err != nil {
		t.Fatal(err)
	}
	if read.StartLine != 1 || read.EndLine != 3 || read.NextOffset != 18 || !read.StartPartial || !read.EndPartial || !regexp.MustCompile(`2#[0-9a-f]{8} \| second`).MatchString(read.Content) || strings.Contains(read.Content, "1#") || strings.Contains(read.Content, "3#") {
		t.Fatalf("read=%+v", read)
	}
	edit := anchoredEdit(t, executor, "a", 2, 2, "new\n")
	op, err = executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory, Permission: policy.AllowAll}, edit)
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
