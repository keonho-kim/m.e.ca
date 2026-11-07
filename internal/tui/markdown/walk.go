// Purpose: Visit rendered logical rows without constructing a full-document row array.
// Pattern: Ordered renderer sink with caller-controlled early stop.
// Usage: Oversized viewport caches build sparse indexes and materialize only requested windows.
// Related: internal/tui/markdown/render_rows.go, internal/tui/viewport/markdown_walk.go.
package markdown

import "strings"

// Walk returns false when rendering requires the same literal fallback as Render.
// A visitor returning false stops successfully and releases all parser scratch.
// Row text and spans are borrowed immutable values.
func Walk(text string, width int, visit func(Line) bool) bool {
	if width < 1 {
		return true
	}
	if len(text) > MaxParseBytes {
		for part := range strings.SplitSeq(SafeText(text), "\n") {
			if !visit(line([]Span{{Text: part}})) {
				break
			}
		}
		return true
	}
	r := renderer{source: []byte(text), width: width, sink: &rowSink{send: visit}}
	document := documentParser.Parse(r.source)
	r.blocks(document, "", 0)
	if r.failed {
		return false
	}
	if !r.stopped {
		for count := len(text) - len(strings.TrimRight(text, "\n")); count > 0; count-- {
			if !visit(Line{}) {
				break
			}
		}
	}
	return true
}
