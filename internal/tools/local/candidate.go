// Purpose: Freeze file-specific change content and approval previews before user approval.
// Pattern: Change candidate preparation.
// Usage: Prepare calls prepareChange for writes and edits after access classification.
// Related: internal/tools/local/prepare.go, internal/tools/local/approval_candidate.go.
package local

import (
	"crypto/sha256"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/textedit"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
)

func (op *operation) prepareChange(call toolapi.Call) error {
	var err error
	op.original, err = snapshot(op.source)
	if err != nil {
		return err
	}
	if op.spec.Name != "write_file" && !op.original.exists {
		return fmt.Errorf("path not found")
	}

	if op.original.exists && op.original.info.IsDir() {
		return fmt.Errorf("target is a directory")
	}
	var before []byte
	if op.original.exists {
		before, err = op.readSource()
		if err != nil {
			return err
		}
	}
	if op.original.exists && sha256.Sum256(before) != op.original.digest {
		return &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "source changed during candidate preparation", Next: "Inspect the current source and prepare a fresh change."}
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
	return op.freezeCandidate(call, before, op.replacement)
}
