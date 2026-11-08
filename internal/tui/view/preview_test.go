// Purpose: Verify only actual diff change rows retain emphasis through wrapping.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/view -run TestPreviewChangeClassification.
// Related: internal/tui/view/preview.go, internal/tui/session/approval_test.go.
package view

import (
	"context"
	"testing"
)

func TestPreviewChangeClassification(t *testing.T) {
	preview := "--- file\n+++ file\n@@ -1,1 +1,1 @@\n---content\n+++content\n context\n"
	for _, isDiff := range []bool{true, false} {
		layout, err := BuildPreview(context.Background(), preview, 80, isDiff)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range layout.Window(0, layout.Count()) {
			var want byte
			if isDiff && (row.Text == "---content" || row.Text == "+++content") {
				want = row.Text[0]
			}
			if row.Change != want {
				t.Fatalf("row %q: change=%q, want %q", row.Text, row.Change, want)
			}
		}
	}
}
