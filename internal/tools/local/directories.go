// Purpose: Freeze recursive directory targets and their existing identities before approval.
// Pattern: Filesystem preparation adapter.
// Usage: Prepare and parallel scheduling share directoryTargets.
// Related: internal/tools/local/directory_execution.go, internal/tools/local/paths.go.
package local

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

type directoryTarget struct {
	target target
	info   os.FileInfo
}

// Include existing ancestors for identity checks; missing ancestors also define
// the write footprint, preventing parallel calls from racing to create a parent.
func (e *Executor) directoryTargets(settings policy.Settings, paths []string) ([]directoryTarget, error) {
	var dirs []directoryTarget
	seen := map[string]bool{}
	var visit func(string) error
	visit = func(path string) error {
		t, err := e.writeTarget(settings.Scope, path)
		if err != nil {
			return err
		}
		if seen[t.absolute] {
			return nil
		}
		info, err := t.root.Stat(t.relative)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && !info.IsDir() {
			return fmt.Errorf("directory path is a file: %s", t.absolute)
		}
		if info == nil {
			if err := visit(filepath.Dir(t.absolute)); err != nil {
				return err
			}
		}
		if len(dirs) >= 64 {
			return fmt.Errorf("directory operation exceeds 64 targets including parents")
		}
		seen[t.absolute] = true
		dirs = append(dirs, directoryTarget{t, info})
		return nil
	}
	for _, path := range paths {
		if strings.TrimSpace(path) == "" || strings.ContainsAny(path, "\r\n") {
			return nil, fmt.Errorf("each directory path must be one nonempty literal path")
		}
		if err := visit(path); err != nil {
			return nil, err
		}
	}
	return dirs, nil
}

func (op *operation) prepareDirectories(call toolapi.Call) error {
	dirs, err := op.executor.directoryTargets(op.settings, op.input.Paths)
	if err != nil {
		return err
	}
	op.directories = dirs
	for _, path := range op.input.Paths {
		target, err := op.executor.writeTarget(op.settings.Scope, path)
		if err != nil {
			return err
		}
		op.directoryPaths = append(op.directoryPaths, target.absolute)
	}
	var names []string
	var files []toolapi.FileDelta
	for _, dir := range dirs {
		if dir.info != nil {
			continue
		}
		names = append(names, dir.target.absolute)
		files = append(files, toolapi.FileDelta{Path: dir.target.absolute, BeforeDigest: "missing", AfterDigest: "directory"})
	}
	op.approval.Target = strings.Join(op.input.Paths, ", ")
	op.approval.Preview = "Create directories (including missing parents):\n" + strings.Join(names, "\n")
	if len(names) == 0 {
		op.approval.Preview = "All requested directories already exist."
	}
	candidate := toolapi.BindCandidate(call, files)
	op.approval.Candidate = &candidate
	return nil
}
