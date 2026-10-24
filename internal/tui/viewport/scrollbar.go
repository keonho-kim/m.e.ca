// Purpose: Map transcript positions to scrollbar geometry and drag positions back to transcript rows.
// Pattern: Pure layout calculation.
// Usage: Session interaction and the view renderer share NewScrollbar geometry.
// Related: internal/tui/viewport/window.go, internal/tui/view/scrollbar.go, internal/tui/session/scrollbar.go.
package viewport

type Scrollbar struct {
	Height, ThumbTop, ThumbRows, MaxStart int
}

func NewScrollbar(total, height, start int) Scrollbar {
	if height <= 0 {
		return Scrollbar{}
	}
	bar := Scrollbar{Height: height, ThumbRows: height, MaxStart: max(0, total-height)}
	if bar.MaxStart == 0 {
		return bar
	}
	bar.ThumbRows = max(1, int(int64(height)*int64(height)/int64(total)))
	travel := height - bar.ThumbRows
	bar.ThumbTop = int((int64(max(0, min(start, bar.MaxStart)))*int64(travel) + int64(bar.MaxStart)/2) / int64(bar.MaxStart))
	return bar
}

func (bar Scrollbar) Visible() bool { return bar.Height > 0 && bar.MaxStart > 0 }

func (bar Scrollbar) Index(row, grab int) int {
	travel := bar.Height - bar.ThumbRows
	if travel <= 0 {
		return 0
	}
	top := max(0, min(travel, row-grab))
	return int((int64(top)*int64(bar.MaxStart) + int64(travel)/2) / int64(travel))
}
