// Purpose: Freeze file-specific change content and approval previews before review.
// Pattern: Change candidate preparation.
// Usage: Prepare calls prepareChange only after Scope and Permission classification.
// Related: internal/tools/local/prepare.go, internal/tools/local/review.go.
package local

import (
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/tools/local/diff"
)

func (op *operation) prepareChange(call toolapi.Call) error {
	var err error
	op.original, err = snapshot(op.source)
	if err != nil {
		return err
	}
	if op.spec.name != "write_file" && op.spec.name != "create_directory" && !op.original.exists {
		return fmt.Errorf("Path not found")
	}
	switch op.spec.name {
	case "create_directory":
		if op.original.exists {
			return fmt.Errorf("directory target already exists")
		}
		op.approval.Preview = "Create empty directory " + op.source.absolute
		if err := op.freezeCandidate(call, nil, nil); err != nil {
			return err
		}
	case "write_file", "edit_file":
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
		op.replacement = []byte(op.input.Content)
		if op.spec.name == "edit_file" {
			replacement, span, err := op.executor.reads.Replace(op.source.absolute, before, op.input.StartAnchor, op.input.EndAnchor, op.input.NewText)
			if err != nil {
				return err
			}
			op.replacement = replacement
			op.editSpan = &span
		}
		op.approval.Preview = op.executor.redact(diff.Preview(op.source.absolute, before, op.replacement, op.original.exists, true))
		op.approval.Diff = true
		if err := op.freezeCandidate(call, before, op.replacement); err != nil {
			return err
		}
	case "move_file":
		op.destination, err = op.executor.resolve(op.settings.Scope, op.input.Destination)
		if err != nil {
			return err
		}
		op.destinationVersion, err = snapshot(op.destination)
		if err != nil {
			return err
		}
		if op.destinationVersion.exists {
			return fmt.Errorf("destination already exists")
		}
		op.approval.Preview = "Move to " + op.destination.absolute
	case "delete_file":
		op.approval.Preview = "Delete " + op.source.absolute
		if !op.original.info.IsDir() {
			before, err := op.readSource()
			if err != nil {
				return err
			}
			op.approval.Preview = op.executor.redact(diff.Preview(op.source.absolute, before, nil, true, false))
			op.approval.Diff = true
			if err := op.freezeCandidate(call, before, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
