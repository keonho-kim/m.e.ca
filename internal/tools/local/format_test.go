// Purpose: Verify format metadata and frozen approval content match the bytes written.
// Pattern: File tool integration contract tests.
// Usage: go test ./internal/tools/local -run TestFormat.
// Related: internal/tools/local/candidate.go, internal/tools/local/file/read.go.
package local

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatLastLineIsPreservedBeforeApproval(t *testing.T) {
	executor, root := testExecutor(t)
	path := filepath.Join(root, "a")
	if err := os.WriteFile(path, []byte("old\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	edit := anchoredEdit(t, executor, "a", 1, 1, "new")
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory, Permission: policy.Approve}, edit)
	if err != nil {
		t.Fatal(err)
	}
	delta := op.Approval().Candidate.Files[0]
	if !delta.BeforeFinalNewline || !delta.AfterFinalNewline {
		t.Fatalf("%+v", delta)
	}
	if _, err := op.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "new\r\n" {
		t.Fatalf("%q %v", got, err)
	}
}
func TestFormatReadShowsTrueEOF(t *testing.T) {
	for _, body := range []string{"a\r\n", "a", ""} {
		executor, root := testExecutor(t)
		if err := os.WriteFile(filepath.Join(root, "a"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.Directory, Permission: policy.Approve}, call("read_file", `{"path":"a"}`))
		if err != nil {
			t.Fatal(err)
		}
		result, err := op.Execute(context.Background(), false)
		if err != nil {
			t.Fatal(err)
		}
		var read struct {
			FinalNewline     bool
			RangeLineEndings string
		}
		if err := json.Unmarshal(result.Content, &read); err != nil {
			t.Fatal(err)
		}
		if read.FinalNewline != (body == "a\r\n") {
			t.Fatalf("%q: %+v", body, read)
		}
	}
}
func TestFormatLiteralChangeRequiresExplicitArgument(t *testing.T) {
	executor, root := testExecutor(t)
	path := filepath.Join(root, "a")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Scope: policy.Directory, Permission: policy.Approve}
	if _, err := executor.Prepare(context.Background(), settings, call("write_file", `{"path":"a","content":"new\n"}`)); err == nil {
		t.Fatal("implicit format change accepted")
	}
	op, err := executor.Prepare(context.Background(), settings, call("write_file", `{"path":"a","content":"new\n","format_policy":"literal"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(op.Approval().Preview, "final newline: false -> true") {
		t.Fatal("format change missing from approval")
	}
	if !op.Approval().Candidate.Files[0].AfterFinalNewline {
		t.Fatal("candidate lost explicit EOF change")
	}
	if _, err := op.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "new\n" {
		t.Fatalf("%q %v", got, err)
	}
}
