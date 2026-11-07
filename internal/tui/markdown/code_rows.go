// Purpose: Seek normalized code rows without retaining a rendered row for every source line.
// Pattern: Immutable code body with sparse byte offsets and shared native rendering.
// Usage: Resumable plans seek every 256 rows; renderer.block shares codeLine's exact styles.
// Related: internal/tui/markdown/plan.go, internal/tui/markdown/render.go.
package markdown

import (
	"strings"
	"unsafe"

	"github.com/yuin/goldmark/v2/ast"
)

const codeMarker = "│ "
const codeSeekRows = 256

type codeRows struct {
	body           string
	points         []int
	heading        Line
	withHeading    bool
	rows, retained int
}

func codeLine(prefix, text string) Line {
	return line(add(add(nil, prefix, Dim), SafeText(text), 0))
}

func newCodeRows(text string, source []byte, node *ast.CodeBlock, budget int) *codeRows {
	body := strings.TrimSuffix(node.Value.Str(source), "\n")
	value := &codeRows{}
	owned := 0
	if body != "" {
		start := blockStart(node, source)
		until := min(len(text), start+len(body)+4096)
		if at := strings.Index(text[start:until], body); at >= 0 {
			body = text[start+at : start+at+len(body)]
		} else {
			if len(body) > budget {
				return nil
			}
			body = strings.Clone(body)
			owned = len(body)
		}
	}
	value.body = body
	value.rows = strings.Count(body, "\n") + 1
	value.points = make([]int, 0, (value.rows+codeSeekRows-1)/codeSeekRows)
	index, offset := 0, 0
	for part := range strings.SplitSeq(body, "\n") {
		if index%codeSeekRows == 0 {
			value.points = append(value.points, offset)
		}
		index++
		offset += len(part) + 1
	}
	if language, ok := node.Language(source); ok {
		value.withHeading = true
		value.heading = line(add(nil, SafeText(language), Dim))
		value.rows++
	}
	value.retained = owned + int(unsafe.Sizeof(*value)) + cap(value.points)*int(unsafe.Sizeof(int(0))) + value.heading.RetainedBytes()
	if value.retained > budget {
		return nil
	}
	return value
}

func (value *codeRows) walk(start int, visit func(Line) bool) bool {
	if value.withHeading {
		if start == 0 {
			if !visit(value.heading) {
				return false
			}
		}
		start = max(0, start-1)
	}
	point := start / codeSeekRows
	if point >= len(value.points) {
		return true
	}
	index := point * codeSeekRows
	for part := range strings.SplitSeq(value.body[value.points[point]:], "\n") {
		if index >= start && !visit(codeLine(codeMarker, part)) {
			return false
		}
		index++
	}
	return true
}
