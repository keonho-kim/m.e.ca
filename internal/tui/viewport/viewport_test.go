// Purpose: Verify event-relative scroll anchors survive appends and reflow.
// Pattern: Presentation behavior tests.
// Usage: go test ./internal/tui/viewport.
// Related: internal/tui/viewport/viewport.go.
package viewport

import (
	"testing"
)

func TestScrollAnchor(t *testing.T) {
	v := View{Follow: true}
	for i := 0; i < 20; i++ {
		v.Append("abcdef", false)
	}
	v.Scroll(-8, 20, 5)
	id := v.AnchorID
	v.Append("new", false)
	lines := v.Lines(3)
	start := v.Start(lines, 5)
	if lines[start].ID != id || v.Follow {
		t.Fatal("reading position moved")
	}
	v.Scroll(1000, 3, 5)
	if !v.Follow {
		t.Fatal("did not resume following")
	}
}
