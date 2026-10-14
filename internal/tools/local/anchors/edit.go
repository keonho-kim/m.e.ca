// Purpose: Validate a complete previously read range before replacing it and reporting new line positions.
// Pattern: Snapshot-bound edit transformation.
// Usage: File preparation calls Replace before Conclave and user approval.
// Related: internal/tools/local/anchors/store.go, internal/tools/local/prepare.go.
package anchors

import (
	"crypto/sha256"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"unicode/utf8"
)

func (s *Store) Replace(path string, current []byte, start, end, replacement string) ([]byte, toolapi.LineChange, error) {
	beginAnchor, err := toolapi.ParseAnchor(start)
	if err != nil {
		return nil, toolapi.LineChange{}, err
	}
	endAnchor, err := toolapi.ParseAnchor(end)
	if err != nil {
		return nil, toolapi.LineChange{}, err
	}
	first, last := beginAnchor.Line, endAnchor.Line
	start, end = strings.TrimSpace(start), strings.TrimSpace(end)
	if last < first || !utf8.ValidString(replacement) {
		return nil, toolapi.LineChange{}, fmt.Errorf("invalid edit range or replacement encoding")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var original, leading string
	var prefix [32]byte
	var snapshotFirst int
	matches := 0
	for _, read := range s.reads {
		begin, finish := first-read.first, last-read.first
		if read.path != path || begin < 0 || finish >= len(read.lines) {
			continue
		}
		if read.anchor(begin) != start || read.anchor(finish) != end {
			continue
		}
		original = strings.Join(read.lines[begin:finish+1], "")
		leading = strings.Join(read.lines[:begin], "")
		prefix = read.prefix
		snapshotFirst = read.first
		matches++
	}
	if matches != 1 {
		return nil, toolapi.LineChange{}, &toolapi.Failure{Code: toolapi.AnchorUnavailable, Detail: "anchors are expired, ambiguous, from different reads, or bound to another file", Next: "Read one range containing both endpoints and copy its new anchors. No edit was applied."}
	}
	lines := split(string(current))
	if last > len(lines) {
		return nil, toolapi.LineChange{}, stale()
	}
	if sha256.Sum256([]byte(strings.Join(lines[:snapshotFirst-1], ""))) != prefix || strings.Join(lines[snapshotFirst-1:first-1], "") != leading {
		return nil, toolapi.LineChange{}, stale()
	}
	actual := strings.Join(lines[first-1:last], "")
	// Short labels locate a snapshot, never authorize a write. Compare the whole
	// range, including intermediate lines, whitespace and original line endings.
	if sha256.Sum256([]byte(actual)) != sha256.Sum256([]byte(original)) || actual != original {
		return nil, toolapi.LineChange{}, stale()
	}
	if last < len(lines) && replacement != "" && !strings.HasSuffix(replacement, "\n") {
		return nil, toolapi.LineChange{}, &toolapi.Failure{Code: toolapi.InvalidArgument, Detail: "replacement must end with a newline before the following untouched line", Next: "Preserve the intended line ending, or include the following line in the edit range."}
	}
	if strings.Contains(original, "\r\n") && !strings.Contains(strings.ReplaceAll(original, "\r\n", ""), "\n") {
		replacement = strings.ReplaceAll(strings.ReplaceAll(replacement, "\r\n", "\n"), "\n", "\r\n")
	}
	span := toolapi.LineChange{OldStartLine: first, OldEndLine: last, NewStartLine: first, NewEndLine: first + len(split(replacement)) - 1, Removed: replacement == ""}
	if span.Removed {
		span.NewStartLine = 0
		span.NewEndLine = 0
	}
	after := strings.Join(lines[:first-1], "") + replacement + strings.Join(lines[last:], "")
	return []byte(after), span, nil
}
func stale() error {
	return &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "the selected range moved or changed since read_file", Next: "Read the current lines again and regenerate the edit anchors; no change was applied."}
}
