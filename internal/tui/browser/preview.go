// Purpose: Own preview scroll state and align numbered source rows at the current pane width.
// Pattern: Pure presentation state.
// Usage: File preview events set the snapshot; renderers and keys use Rows to bound scrolling.
// Related: internal/tui/session/file_preview.go, internal/tui/viewport/viewport.go.
package browser

import (
	"fmt"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
	"strconv"
	"strings"
)

type Preview struct {
	Path, Content, Error     string
	Full, Loading, Truncated bool
	Offset                   int
}

func (p Preview) Rows(width int) []string {
	if p.Loading {
		return []string{"파일 읽는 중…"}
	}
	if p.Error != "" {
		return viewport.Wrap(p.Error, max(1, width))
	}
	if p.Content == "" {
		return []string{"(빈 파일)"}
	}
	lines := strings.Split(strings.TrimSuffix(p.Content, "\n"), "\n")
	digits := max(3, len(strconv.Itoa(len(lines))))
	gutter := digits + 3
	var rows []string
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if width <= gutter {
			rows = append(rows, viewport.Wrap(fmt.Sprintf("%*d | %s", digits, i+1, line), max(1, width))...)
			continue
		}
		for j, part := range viewport.Wrap(line, width-gutter) {
			prefix := strings.Repeat(" ", gutter)
			if j == 0 {
				prefix = fmt.Sprintf("%*d | ", digits, i+1)
			}
			rows = append(rows, prefix+part)
		}
	}
	if p.Truncated {
		rows = append(rows, "[미리보기는 처음 16MiB까지만 표시]")
	}
	return rows
}
