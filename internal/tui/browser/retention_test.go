// Purpose: Verify navigating away releases discarded directory listings.
// Pattern: Memory ownership regression test.
// Usage: go test ./internal/tui/browser -run TestMoveReleasesColumns.
// Related: internal/tui/browser/browser.go.
package browser

import (
	"github.com/keonho-kim/meca/internal/tools/local"
	"testing"
)

func TestMoveReleasesColumns(t *testing.T) {
	b := Browser{Columns: []Column{{Path: "root"}, {Path: "child", Entries: []local.Entry{{Name: "retained"}}}}}
	b.Move(0)
	for _, c := range b.Columns[:cap(b.Columns)][len(b.Columns):] {
		if c.Entries != nil || c.Path != "" {
			t.Fatal("discarded directory listing retained")
		}
	}
}
