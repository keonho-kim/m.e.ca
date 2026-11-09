// Purpose: Freeze file-specific change content and approval previews before user approval.
// Pattern: Change candidate preparation.
// Usage: Prepare calls prepareChange for writes and edits after access classification.
// Related: internal/tools/local/prepare.go, internal/tools/local/approval_candidate.go.
package local

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/local/textedit"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
)

func (op *fileOperation) prepareChange(call toolapi.Call) error {
	var err error
	var before []byte
	op.original, before, err = capture(op.source, true)
	if err != nil {
		return err
	}
	if op.spec.Name != "write_file" && !op.original.exists {
		return faults.Errorf(diagcatalog.LocalPathNotFound)
	}

	if op.original.exists && op.original.info.IsDir() {
		return faults.Errorf(diagcatalog.LocalTargetIsADirectory)
	}
	op.replacement = []byte(op.input.Content)
	if op.spec.Name == "edit_file" {
		replacement, spans, err := textedit.ReplaceMany(op.source.absolute, before, op.input.Edits, op.input.FormatPolicy)
		if err != nil {
			return err
		}
		op.replacement = replacement
		op.editSpans = spans
	}
	if op.spec.Name == "write_file" {
		op.replacement, err = textformat.Preserve(before, op.replacement, op.input.FormatPolicy)
		if err != nil {
			return err
		}
	}
	if err := op.freezeCandidate(call, before, op.replacement); err != nil {
		return err
	}
	// Execution owns the frozen bytes; decoded arguments are no longer needed.
	op.input.Content, op.input.Edits = "", nil
	return nil
}
