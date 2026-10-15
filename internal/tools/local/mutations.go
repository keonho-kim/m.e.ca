// Purpose: Apply prepared file mutations and return their exact changed-file metadata.
// Pattern: Filesystem mutation Adapter.
// Usage: Prepared create, write, edit, move and delete operations call mutate.
// Related: internal/tools/local/prepare.go, internal/tools/local/dispatch.go.
package local

import (
	"crypto/sha256"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	fileops "github.com/keonho-kim/v-code/internal/tools/local/file"
	resultfmt "github.com/keonho-kim/v-code/internal/tools/result"
	"os"
)

func (op *operation) mutate() (toolapi.Result, error) {
	switch op.spec.name {
	case "write_file", "edit_file":
		mode := os.FileMode(0o644)
		if op.original.exists {
			mode = op.original.info.Mode().Perm()
		}
		if err := fileops.Write(op.source.root, op.source.relative, op.replacement, mode); err != nil {
			return toolapi.Result{}, err
		}
	case "delete_file":
		if err := op.source.root.Remove(op.source.relative); err != nil {
			return toolapi.Result{}, err
		}
	case "move_file":
		current, err := op.executor.resolve(op.settings.Scope, op.input.Destination)
		if err != nil {
			return toolapi.Result{}, err
		}
		version, err := snapshot(current)
		if err != nil {
			return toolapi.Result{}, err
		}
		if current.absolute != op.destination.absolute || !sameVersion(op.destinationVersion, version) {
			return toolapi.Result{}, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "destination changed after preparation", Next: "Inspect the destination and prepare a new move."}
		}
		if err := op.source.root.Rename(op.source.relative, op.destination.relative); err != nil {
			return toolapi.Result{}, err
		}
	default:
		return toolapi.Result{}, fmt.Errorf("unknown file mutation")
	}
	var size *int
	summary := ""
	switch op.spec.name {
	case "write_file", "edit_file":
		bytes := len(op.replacement)
		size = &bytes
		action := "Updated file"
		if !op.original.exists {
			action = "Created file"
		}
		summary = fmt.Sprintf("%s %s (%d bytes)", action, op.source.absolute, bytes)
	case "delete_file":
		summary = "Deleted " + op.source.absolute
	case "move_file":
		summary = "Moved " + op.source.absolute + " to " + op.destination.absolute
	}
	result, err := resultfmt.Encode(struct {
		Action    string              `json:"action"`
		Path      string              `json:"path"`
		Changed   bool                `json:"changed"`
		SizeBytes *int                `json:"size_bytes,omitempty"`
		Lines     *toolapi.LineChange `json:"lines,omitempty"`
	}{op.spec.name, op.source.absolute, true, size, op.editSpan}, summary)
	if err != nil {
		return result, err
	}
	change := toolapi.FileChange{Action: op.spec.name, Path: op.source.absolute, Lines: op.editSpan}
	if op.spec.name == "move_file" {
		change.Destination = op.destination.absolute
	}
	if !op.original.exists {
		change.BeforeDigest = "missing"
	} else if !op.original.info.IsDir() {
		change.BeforeDigest = fmt.Sprintf("%x", op.original.digest)
	}
	if change.BeforeDigest != "" {
		change.AfterDigest = "missing"
		if op.spec.name == "write_file" || op.spec.name == "edit_file" {
			change.AfterDigest = fmt.Sprintf("%x", sha256.Sum256(op.replacement))
		}
	}
	result.Changes = []toolapi.FileChange{change}
	return result, nil
}
