// Purpose: Preserve existing EOF and newline conventions before a change candidate is frozen.
// Pattern: Pure byte transformation with explicit literal override.
// Usage: File writes and exact-text edits call Preserve to keep newline and EOF conventions.
// Related: internal/tools/local/candidate.go, internal/tools/local/textedit/edit.go.
package textformat

import (
	"bytes"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func LineEndings(body []byte) string {
	crlf := bytes.Count(body, []byte("\r\n"))
	lf := bytes.Count(body, []byte("\n")) - crlf
	switch {
	case crlf > 0 && lf > 0:
		return "mixed"
	case crlf > 0:
		return "crlf"
	case lf > 0:
		return "lf"
	default:
		return "none"
	}
}

// Preserve only appends a missing terminator; it never trims blanks or normalizes a whole file.
func Preserve(before, after []byte, policy string) ([]byte, error) {
	if policy == "literal" {
		return after, nil
	}
	if policy != "" && policy != "preserve" {
		return nil, faults.Errorf(diagcatalog.TextformatUnknownFormatPolicy)
	}
	if len(before) == 0 || len(after) == 0 {
		return after, nil
	}
	previous := bytes.HasSuffix(before, []byte("\n"))
	next := bytes.HasSuffix(after, []byte("\n"))
	if !previous && next {
		return nil, faults.Errorf(diagcatalog.TextformatFormatChangeExistingFileHasNoFinal)
	}
	if previous && !next {
		ending := []byte("\n")
		if bytes.HasSuffix(before, []byte("\r\n")) {
			ending = []byte("\r\n")
		}
		after = append(bytes.Clone(after), ending...)
	}
	oldStyle, newStyle := LineEndings(before), LineEndings(after)
	if oldStyle != "none" && newStyle != "none" && oldStyle != newStyle {
		return nil, faults.Errorf(diagcatalog.TextformatFormatChangeNewlineConventionWouldChangeFrom, oldStyle, newStyle)
	}
	return after, nil
}
