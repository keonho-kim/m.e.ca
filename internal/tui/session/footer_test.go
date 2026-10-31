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
	a := screenFixture(t)
	a.selection.Model = "chosen"
	a.output.Append(strings.Repeat("history\n", 50), false)
	for _, size := range [][2]int{{80, 24}, {48, 12}, {32, 8}} {
		a.screen.SetSize(size[0], size[1])
		a.render()
		status := screenLine(a.screen, size[1]-a.footerRows, size[0])
		keys := footerKeyText(a)
		if !strings.Contains(status, "chosen") || strings.Contains(status, "전체 시스템") || strings.Contains(status, "^C") {
			t.Fatalf("status=%q", status)
		}
		if !strings.Contains(keys, "^C") || strings.Contains(keys, "전체 시스템") || strings.Contains(keys, "chosen") {
			t.Fatalf("keys=%q", keys)
		}
		for row := 0; row < size[1]-a.footerRows; row++ {
			if strings.Contains(screenLine(a.screen, row, size[0]), "chosen") {
				t.Fatal("status was rendered above the footer")
			}
		}
	}
}
