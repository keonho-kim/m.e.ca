// Purpose: Verify repeated edits require a check without treating reads as verification.
// Pattern: State transition tests.
// Usage: go test ./internal/agent/progress.
// Related: internal/agent/progress/revisions.go.
package progress

import (
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"testing"
)

func TestRevisionsRequireCheckAfterThreeEdits(t *testing.T) {
	var revisions Revisions
	for i := 0; i < 3; i++ {
		revisions.Observe([]toolapi.FileChange{{Path: "a", BeforeDigest: "old", AfterDigest: "new"}})
	}
	if revisions.Due() != "a" {
		t.Fatal("repeated edits did not require verification")
	}
	revisions.Observe(nil)
	if revisions.Due() != "a" {
		t.Fatal("read cleared verification checkpoint")
	}
	revisions.Checked()
	if revisions.Due() != "" {
		t.Fatal("completed check did not allow corrections")
	}
	for _, p := range []string{"a", "b", "c"} {
		revisions.Observe([]toolapi.FileChange{{Path: p, BeforeDigest: "old", AfterDigest: "new"}})
	}
	if revisions.Due() != "" {
		t.Fatal("independent files counted as repeated edits")
	}
}
