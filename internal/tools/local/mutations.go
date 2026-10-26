// Purpose: Apply prepared file writes and edits with exact changed-file metadata.
// Pattern: Filesystem mutation Adapter.
// Usage: Prepared write and edit operations call mutate after policy and version checks.
// Related: internal/tools/local/prepare.go, internal/tools/local/dispatch.go.
package local

import (
	"crypto/sha256"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	fileops "github.com/keonho-kim/meca/internal/tools/local/file"
	resultfmt "github.com/keonho-kim/meca/internal/tools/result"
	"os"
)

func (op *operation) mutate() (toolapi.Result, error) {
	mode := os.FileMode(0o644)
	if op.original.exists {
		mode = op.original.info.Mode().Perm()
	}
	if err := fileops.Write(op.source.root, op.source.relative, op.replacement, mode); err != nil {
		return toolapi.Result{}, err
	}
	action := "Updated file"
	if !op.original.exists {
		action = "Created file"
	}
	size := len(op.replacement)
	summary := fmt.Sprintf("%s %s (%d bytes)", action, op.source.absolute, size)
	result, err := resultfmt.Encode(struct {
		Action    string               `json:"action"`
		Path      string               `json:"path"`
		Changed   bool                 `json:"changed"`
		SizeBytes int                  `json:"size_bytes"`
		Lines     []toolapi.LineChange `json:"lines,omitempty"`
	}{op.spec.Name, op.source.absolute, true, size, op.editSpans}, summary)
	if err != nil {
		return result, err
	}
	change := toolapi.FileChange{Action: op.spec.Name, Path: op.source.absolute, Lines: op.editSpans}
	change.BeforeDigest = versionValue(op.original)
	change.AfterDigest = fmt.Sprintf("%x", sha256.Sum256(op.replacement))
	result.Versions = map[string]string{change.Path: change.AfterDigest}
	result.Changes = []toolapi.FileChange{change}
	return result, nil
}
