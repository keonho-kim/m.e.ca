// Purpose: Apply disjoint exact-text replacements against one current source and report positions.
// Pattern: Atomic in-memory batch transformation.
// Usage: File preparation calls ReplaceMany before candidate freezing and user approval.
// Related: internal/tools/local/textedit/edit.go, internal/tools/local/candidate.go.
package textedit

import (
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/tools/local/textformat"
	"sort"
	"strings"
)

func ReplaceMany(path string, current []byte, edits []Edit, format string) ([]byte, []toolapi.LineChange, error) {
	if len(edits) < 1 || len(edits) > 64 {
		return nil, nil, fmt.Errorf("edits requires 1 to 64 replacements")
	}
	source := string(current)
	ranges := make([]replacement, len(edits))
	for i, edit := range edits {
		r, err := resolve(source, edit, format)
		if err != nil {
			f := toolapi.FailureOf(err)
			f.Detail = fmt.Sprintf("%s edits[%d]: %s", path, i, f.Detail)
			return nil, nil, f
		}
		r.index = i
		ranges[i] = r
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	for i := 1; i < len(ranges); i++ {
		if ranges[i].start < ranges[i-1].end {
			return nil, nil, fmt.Errorf("%s edits[%d] overlaps edits[%d]; nothing was applied", path, ranges[i-1].index, ranges[i].index)
		}
	}
	var out strings.Builder
	spans := make([]toolapi.LineChange, len(ranges))
	cursor, line := 0, 1
	for _, r := range ranges {
		gap := source[cursor:r.start]
		out.WriteString(gap)
		line += strings.Count(gap, "\n")
		oldStart := 1 + strings.Count(source[:r.start], "\n")
		oldEnd := oldStart + strings.Count(source[r.start:r.end], "\n")
		if strings.HasSuffix(source[r.start:r.end], "\n") {
			oldEnd--
		}
		n := strings.Count(r.text, "\n")
		end := line + n
		if strings.HasSuffix(r.text, "\n") {
			end--
		}
		span := toolapi.LineChange{OldStartLine: oldStart, OldEndLine: oldEnd, NewStartLine: line, NewEndLine: end, Removed: r.text == "" && (r.start == 0 || source[r.start-1] == '\n') && (r.end == len(source) || source[r.end-1] == '\n')}
		if span.Removed {
			span.NewStartLine = 0
			span.NewEndLine = 0
		}
		spans[r.index] = span
		out.WriteString(r.text)
		line += n
		cursor = r.end
	}
	out.WriteString(source[cursor:])
	after, err := textformat.Preserve(current, []byte(out.String()), format)
	return after, spans, err
}
