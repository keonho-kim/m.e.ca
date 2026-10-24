// Purpose: Declare the single-file, multiple-range native edit contract.
// Pattern: Declarative tool schema.
// Usage: The fixed catalog includes editSpecification in its native definitions.
// Related: internal/tools/local/catalog.go, internal/tools/local/textedit/ranges.go.
package local

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func editSpecification() specification {
	return specification{"edit_file", "Editing", "Replace disjoint ranges in one UTF-8 file within allowed write paths against the same original source. Put related changes in one edits array; every range is checked before anything is applied. old_text must match the current file exactly once. Overlapping ranges or files changed since preparation are rejected. Returns final line positions for every range. Preserve is the default newline/EOF policy.", []toolapi.Parameter{
		filePath(),
		{Name: "edits", Kind: toolapi.ObjectList, Required: true, Max: 64, Description: "One to 64 non-overlapping ranges against the original file, never against an earlier replacement in this call.", Fields: []toolapi.Parameter{
			{Name: "old_text", Kind: toolapi.Text, Required: true, MaxBytes: toolapi.MaxArgumentsBytes, Description: "Exact nonempty current source, typically inspected through execute_command. Include enough context to match once; preserve whitespace."},
			textField("new_text", "Exact source replacement without line/hash labels or wrapping fences. Empty deletes the range.", true),
		}},
	}, policy.Change}
}
