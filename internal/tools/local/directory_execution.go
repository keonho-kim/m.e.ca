// Purpose: Apply approved recursive directory creation and preserve partial execution facts.
// Pattern: Filesystem mutation adapter.
// Usage: Prepared directory operations invoke createDirectories after policy and approval checks.
// Related: internal/tools/local/directories.go, internal/tools/local/operation.go.
package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	resultfmt "github.com/keonho-kim/v-code/internal/tools/result"
)

func (op *operation) createDirectories(ctx context.Context) (toolapi.Result, error) {
	current, err := op.executor.directoryTargets(op.settings, op.input.Paths)
	if err != nil {
		return toolapi.Result{}, err
	}
	if len(current) != len(op.directories) {
		return toolapi.Result{}, directoryStale()
	}
	for i, dir := range current {
		old := op.directories[i]
		if dir.target.absolute != old.target.absolute || (dir.info == nil) != (old.info == nil) ||
			dir.info != nil && !os.SameFile(dir.info, old.info) {
			return toolapi.Result{}, directoryStale()
		}
	}
	var created []string
	var changes []toolapi.FileChange
	identities := map[string]os.FileInfo{}
	finish := func(cause error) (toolapi.Result, error) {
		result, encodeErr := resultfmt.Encode(struct {
			Action  string   `json:"action"`
			Changed bool     `json:"changed"`
			Created []string `json:"created"`
			Paths   []string `json:"paths"`
		}{"create_directory", len(created) > 0, created, op.directoryPaths},
			fmt.Sprintf("Created directories: %d; requested paths: %d", len(created), len(op.input.Paths)))
		result.Changes = changes
		result.Failed = cause != nil
		if cause == nil {
			result.Versions = map[string]string{}
			for _, path := range op.directoryPaths {
				result.Versions[path] = "directory"
			}
		}
		return result, errors.Join(cause, encodeErr)
	}
	for _, dir := range current {
		if err := ctx.Err(); err != nil {
			return finish(err)
		}
		if dir.info != nil {
			identities[dir.target.absolute] = dir.info
			continue
		}
		// Open the parent itself so a concurrent path substitution cannot redirect
		// Mkdir through a newly introduced symlink.
		parentPath := filepath.Dir(dir.target.absolute)
		parentTarget, err := op.executor.resolve(op.settings.Scope, parentPath)
		if err != nil {
			return finish(err)
		}
		if parentTarget.absolute != parentPath {
			return finish(fmt.Errorf("directory parent changed: %s", parentPath))
		}
		parent, err := parentTarget.root.OpenRoot(parentTarget.relative)
		if err != nil {
			return finish(err)
		}
		info, statErr := parent.Stat(".")
		if statErr != nil || identities[parentPath] == nil || !os.SameFile(info, identities[parentPath]) {
			return finish(errors.Join(fmt.Errorf("directory parent identity changed: %s", parentPath), statErr, parent.Close()))
		}
		name := filepath.Base(dir.target.absolute)
		err = parent.Mkdir(name, 0755)
		if err == nil {
			created = append(created, dir.target.absolute)
			changes = append(changes, toolapi.FileChange{Action: "create_directory", Path: dir.target.absolute, BeforeDigest: "missing", AfterDigest: "directory"})
			identities[dir.target.absolute], err = parent.Stat(name)
		}
		closeErr := parent.Close()
		if err != nil || closeErr != nil {
			return finish(errors.Join(err, closeErr))
		}
	}
	return finish(nil)
}
func directoryStale() error {
	return &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "directory targets changed after preparation", Next: "Inspect the current directory paths and retry with the intended paths."}
}
