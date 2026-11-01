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

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

type directoryTarget struct {
	target target
	info   os.FileInfo
}

// Include existing ancestors for identity checks; missing ancestors also define
// the write footprint, preventing parallel calls from racing to create a parent.
func (e *Executor) directoryTargets(settings policy.Settings, paths []string) ([]directoryTarget, error) {
	dirs, _, err := e.planDirectories(settings, paths)
	return dirs, err
}

func (e *Executor) planDirectories(settings policy.Settings, paths []string) (dirs []directoryTarget, requested []target, err error) {
	if err := e.boundary.Check(); err != nil {
		return nil, nil, err
	}
	defer func() { err = errors.Join(err, e.boundary.Check()) }()
	return e.collectDirectories(settings, paths)
}

// The planning owner checks the project at both ends. Each individual path is
// still canonicalized so aliases and private paths never use a cached decision.
func (e *Executor) collectDirectories(settings policy.Settings, paths []string) ([]directoryTarget, []target, error) {
	var dirs []directoryTarget
	var requested []target
	seen := map[string]bool{}
	var visit func(target) error
	visit = func(t target) error {
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
			parent, err := e.resolveTarget(settings.Scope, filepath.Dir(t.absolute))
			if err != nil {
				return err
			}
			if err := visit(parent); err != nil {
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
			return nil, nil, fmt.Errorf("each directory path must be one nonempty literal path")
		}
		t, err := e.resolveTarget(settings.Scope, path)
		if err != nil {
			return nil, nil, err
		}
		if err := visit(t); err != nil {
			return nil, nil, err
		}
		requested = append(requested, t)
	}
	return dirs, requested, nil
}

func (op *directoryOperation) prepareDirectories(call toolapi.Call) error {
	dirs, requested, err := op.executor.planDirectories(op.settings, op.input.Paths)
	if err != nil {
		return err
	}
	op.directories = dirs
	for _, target := range requested {
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
