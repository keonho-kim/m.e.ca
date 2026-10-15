// Purpose: Verify exact frozen changes and zero-line creation exceptions retain approval and stale guards.
// Pattern: Filesystem contract tests.
// Usage: go test ./internal/tools/local -run TestReviewCandidate.
// Related: internal/tools/local/review.go, internal/tools/local/prepare.go, internal/tools/local/mutations.go.
package local

import (
	"context"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"testing"
)

func TestReviewCandidateCountsActualChanges(t *testing.T) {
	executor, root := testExecutor(t)
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("same\nold\ntail\n"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := policy.Settings{Permission: policy.Approve, Scope: policy.Directory}
	for _, tc := range []struct {
		name, args string
		lines      int
	}{
		{"edit_file", string(anchoredEdit(t, executor, "a", 2, 2, "new\n").Arguments), 2},
		{"write_file", `{"path":"a","content":""}`, 3},
		{"write_file", `{"path":"empty","content":""}`, 0},
		{"delete_file", `{"path":"a"}`, 3},
		{"create_directory", `{"paths":["folder"]}`, 0},
	} {
		op, err := executor.Prepare(context.Background(), settings, call(tc.name, tc.args))
		if err != nil {
			t.Fatal(err)
		}
		candidate := op.Approval().Candidate
		if candidate == nil || candidate.Lines() != tc.lines || !op.Approval().Required {
			t.Fatalf("%s: %+v", tc.name, op.Approval())
		}
		if err := candidate.Check(call(tc.name, tc.args)); err != nil {
			t.Fatal(err)
		}
	}
}
func TestReviewCandidateEmptyDirectoryStillNeedsApproval(t *testing.T) {
	executor, root := testExecutor(t)
	op, err := executor.Prepare(context.Background(), policy.Settings{Permission: policy.Approve, Scope: policy.Directory}, call("create_directory", `{"paths":["folder"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := op.Execute(context.Background(), false); err == nil {
		t.Fatal("unapproved directory created")
	}
	if _, err := op.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(root, "folder"))
	if err != nil || !info.IsDir() {
		t.Fatalf("missing directory: %v", err)
	}
}
