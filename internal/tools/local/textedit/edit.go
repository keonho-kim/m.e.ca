// Purpose: Resolve exact-text edits against the current source.
// Pattern: Pure exact-text matching.
// Usage: ReplaceMany resolves every replacement before freezing any file mutation.
// Related: internal/tools/local/textedit/ranges.go, internal/tools/local/candidate.go.
package textedit

import (
	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Edit struct {
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}
type replacement struct {
	start, end, index int
	text              string
}

func resolve(current string, edit Edit, format string) (replacement, error) {
	if edit.OldText == "" || !utf8.ValidString(edit.OldText) || !utf8.ValidString(edit.NewText) {
		return replacement{}, faults.Errorf(diagcatalog.TexteditOldTextMustBeNonemptyAndBoth)
	}
	start := strings.Index(current, edit.OldText)
	if start < 0 {
		return replacement{}, &faults.Failure{Code: diagcatalog.TextNotFound, Detail: diagcatalog.Text(diagcatalog.TexteditOldTextDoesNotOccurInThe), Next: diagcatalog.Text(diagcatalog.TexteditReadTheAffectedSourceAndProvideIts)}
	}
	// Starting one byte later also detects overlapping duplicate matches.
	if strings.Contains(current[start+1:], edit.OldText) {
		return replacement{}, &faults.Failure{Code: diagcatalog.AmbiguousText, Detail: diagcatalog.Text(diagcatalog.TexteditOldTextOccursMoreThanOnce), Next: diagcatalog.Text(diagcatalog.TexteditIncludeEnoughUnchangedSurroundingSourceToIdentify)}
	}
	text := edit.NewText
	if format != "literal" {
		if strings.Contains(edit.OldText, "\r\n") && !strings.Contains(strings.ReplaceAll(edit.OldText, "\r\n", ""), "\n") {
			text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
		}
		if strings.HasSuffix(edit.OldText, "\n") && start+len(edit.OldText) < len(current) && text != "" && !strings.HasSuffix(text, "\n") {
			if strings.HasSuffix(edit.OldText, "\r\n") {
				text += "\r\n"
			} else {
				text += "\n"
			}
		}
	}
	return replacement{start: start, end: start + len(edit.OldText), text: text}, nil
}
