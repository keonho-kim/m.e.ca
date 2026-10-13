// Purpose: Verify only actual diff change rows retain emphasis through wrapping.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/view -run TestPreviewChangeClassification.
// Related: internal/tui/view/preview.go, internal/tui/session/approval_test.go.
package view

import "testing"

func TestPreviewChangeClassification(t *testing.T) {
	preview := "--- file\n+++ file\n@@ -1,1 +1,1 @@\n---content\n+++content\n context\n"
	for _, isDiff := range []bool{true, false} {
		for _, row := range PreviewLines(preview, 80, isDiff) {
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
