// Purpose: Verify status stays above the bottom keyboard hints during scrolling and resizing.
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
		status := screenLine(a.screen, size[1]-2, size[0])
		keys := screenLine(a.screen, size[1]-1, size[0])
		if !strings.Contains(status, "chosen") || !strings.Contains(status, "프로젝트") || strings.Contains(status, "^C") {
			t.Fatalf("status=%q", status)
		}
		if !strings.Contains(keys, "^C") || strings.Contains(keys, "프로젝트") || strings.Contains(keys, "chosen") {
			t.Fatalf("keys=%q", keys)
		}
		for row := 0; row < size[1]-2; row++ {
			if strings.Contains(screenLine(a.screen, row, size[0]), "프로젝트") {
				t.Fatal("status was rendered above the footer")
			}
		}
	}
}
