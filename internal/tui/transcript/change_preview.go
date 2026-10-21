// Purpose: Project frozen unified changes into aligned literal rows and explicit change ranges.
// Pattern: Pure presentation projection.
// Usage: Successful file results display the persisted preview without reading files again.
// Related: internal/tui/transcript/file_results.go, internal/tui/viewport/changes.go, internal/agent/run/change_preview.go.
package transcript

import (
	"fmt"
	"github.com/keonho-kim/v-code/internal/tui/locale"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
	"regexp"
	"strconv"
	"strings"
)

var changeHunk = regexp.MustCompile(`^@@ -([0-9]+)(?:,[0-9]+)? \+([0-9]+)(?:,[0-9]+)? @@`)

type changeRow struct {
	kind   byte
	number int
	text   string
}

func renderChangePreview(preview string, lang locale.Language) toolOutput {
	var rows []changeRow
	oldLine, newLine, width := 0, 0, 3
	inHunk := false
	for _, line := range strings.Split(strings.TrimSuffix(preview, "\n"), "\n") {
		if match := changeHunk.FindStringSubmatch(line); match != nil {
			old, oldErr := strconv.Atoi(match[1])
			next, nextErr := strconv.Atoi(match[2])
			if oldErr != nil || nextErr != nil {
				return toolOutput{text: indentTool(preview)}
			}
			if inHunk {
				rows = append(rows, changeRow{text: "…"})
			}
			oldLine, newLine, inHunk = old, next, true
			continue
		}
		if !inHunk && (strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ")) {
			continue
		}
		row := changeRow{text: line}
		if inHunk && len(line) > 0 {
			switch line[0] {
			case '+':
				row = changeRow{'+', newLine, line[1:]}
				newLine++
			case '-':
				row = changeRow{'-', oldLine, line[1:]}
				oldLine++
			case ' ':
				row = changeRow{' ', newLine, line[1:]}
				oldLine++
				newLine++
			}
		}
		if line == `\ No newline at end of file` {
			row = changeRow{text: lang.Text("(마지막 줄바꿈 없음)")}
		}
		if line == "No content changes." {
			row = changeRow{text: lang.Text("(내용 변경 없음 / 빈 파일)")}
		}
		if row.number > 0 {
			width = max(width, len(strconv.Itoa(row.number)))
		}
		rows = append(rows, row)
	}
	var out strings.Builder
	var changes []viewport.ChangeRange
	for _, row := range rows {
		start := out.Len()
		if row.number > 0 {
			out.WriteString(indentTool(fmt.Sprintf("%c %*d | %s", row.kind, width, row.number, row.text)))
		} else {
			out.WriteString(indentTool(row.text))
		}
		if row.kind == '+' || row.kind == '-' {
			changes = append(changes, viewport.ChangeRange{Start: start, End: out.Len() - 1, Kind: row.kind})
		}
	}
	return toolOutput{text: out.String(), changes: changes}
}
