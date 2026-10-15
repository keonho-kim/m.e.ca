// Purpose: Bound presentation-only edge spacing when joining transcript and streaming blocks.
// Pattern: Pure line composition.
// Usage: View.Lines and session rendering share AppendLines; model blocks use modelEdges.
// Related: internal/tui/viewport/viewport.go, internal/tui/session/render_chat.go.
package viewport

import "strings"

const maxBoundaryBlanks = 2

func blank(line Line) bool { return strings.TrimSpace(line.Text) == "" }

// AppendLines removes only excess leading separator rows from the incoming block.
// Interior rows and existing literal tool/file output remain untouched.
func AppendLines(previous, next []Line) []Line {
	trailing := 0
	for i := len(previous) - 1; i >= 0 && blank(previous[i]); i-- {
		trailing++
		if trailing >= maxBoundaryBlanks {
			break
		}
	}
	leading := 0
	for leading < len(next) && blank(next[leading]) {
		leading++
	}
	skip := max(0, leading-max(0, maxBoundaryBlanks-trailing))
	return append(previous, next[skip:]...)
}

func modelEdges(lines []Line) []Line {
	for len(lines) > 0 && blank(lines[0]) {
		lines = lines[1:]
	}
	end := len(lines)
	for end > 0 && blank(lines[end-1]) {
		end--
	}
	return lines[:min(len(lines), end+maxBoundaryBlanks)]
}
