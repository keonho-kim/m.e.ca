// Purpose: Resolve exact-text edits against the current source.
// Pattern: Pure exact-text matching.
// Usage: ReplaceMany resolves every replacement before freezing any file mutation.
// Related: internal/tools/local/textedit/ranges.go, internal/tools/local/candidate.go.
package textedit

import (
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"unicode/utf8"
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
		return replacement{}, fmt.Errorf("old_text must be nonempty and both texts must be UTF-8")
	}
	start := strings.Index(current, edit.OldText)
	if start < 0 {
		return replacement{}, &toolapi.Failure{Code: "text_not_found", Detail: "old_text does not occur in the current source", Next: "Read the affected source and provide its exact old_text. Nothing was applied."}
	}
	if strings.LastIndex(current, edit.OldText) != start {
		return replacement{}, &toolapi.Failure{Code: "ambiguous_text", Detail: "old_text occurs more than once", Next: "Include enough unchanged surrounding source to identify one occurrence. Nothing was applied."}
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
