// Purpose: Verify scrollbar bounds, endpoint mapping and drag stability across viewport sizes.
// Pattern: Pure geometry contract tests.
// Usage: go test ./internal/tui/viewport -run TestScrollbar.
// Related: internal/tui/viewport/scrollbar.go.
package viewport

import "testing"

func TestScrollbarGeometryAndDragging(t *testing.T) {
	for _, height := range []int{0, 1, 2, 7, 20, 100} {
		for _, total := range []int{0, 1, 7, 20, 100, 1000000} {
			for _, start := range []int{-10, 0, total / 2, total, total + 10} {
				bar := NewScrollbar(total, height, start)
				if bar.Height != height || bar.ThumbTop < 0 || bar.ThumbRows < 0 || bar.ThumbTop+bar.ThumbRows > height {
					t.Fatalf("invalid geometry: total=%d height=%d start=%d bar=%+v", total, height, start, bar)
				}
				if height > 0 && total > height && (!bar.Visible() || bar.ThumbRows < 1) {
					t.Fatal("overflow has no usable thumb")
				}
				if height < 2 || !bar.Visible() {
					continue
				}
				if bar.Index(-100, 0) != 0 || bar.Index(height+100, 0) != bar.MaxStart {
					t.Fatalf("drag endpoints do not map to transcript endpoints: %+v", bar)
				}
				if bar.Index(bar.ThumbTop+bar.ThumbRows-1, bar.ThumbRows-1) != bar.Index(bar.ThumbTop, 0) {
					t.Fatal("grabbing inside the thumb moves the viewport")
				}
			}
		}
	}
}
