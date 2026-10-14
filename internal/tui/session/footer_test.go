// Purpose: Verify model and policy status stay on one bottom row with keyboard hints during scrolling and resizing.
// Pattern: Terminal layout contract test.
// Usage: go test ./internal/tui/session -run TestFixedFooter.
// Related: internal/tui/session/render.go, internal/tui/view/footer.go.
package session

import (
	"strings"
	"testing"
)

func TestFixedFooterContainsStatusAndKeys(t *testing.T) {
	a := fixture(t)
	a.selection.Model = "chosen"
	a.output.Append(strings.Repeat("history\n", 50), false)
	for _, size := range [][2]int{{80, 24}, {48, 12}, {32, 8}} {
		a.screen.SetSize(size[0], size[1])
		a.render()
		line := screenLine(a.screen, size[1]-1, size[0])
		if !strings.Contains(line, "프로젝트") || !strings.Contains(line, "^C") {
			t.Fatalf("footer=%q", line)
		}
		for row := 0; row < size[1]-1; row++ {
			if strings.Contains(screenLine(a.screen, row, size[0]), "프로젝트") {
				t.Fatal("status was rendered above the footer")
			}
		}
	}
}
