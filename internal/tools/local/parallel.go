// Purpose: Plan independent file waves with one access snapshot per call and bounded preparation memory.
// Pattern: Filesystem scheduling adapter; execution still rechecks policy and versions.
// Usage: The runtime asks ParallelBatch before preparation and rechecks it after approval.
// Related: internal/tools/local/paths.go, internal/agent/run/batch.go.
package local

import (
	"context"
	"errors"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
	"os"
	"path/filepath"
	"strings"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

// This bounds estimated prepared bodies, not accepted input sizes. Large calls
// still execute alone; increasing concurrency must not multiply large snapshots.
const maxParallelBytes int64 = 64 * 1024 * 1024

type parallelTarget struct {
	target
	info os.FileInfo
}

func (e *Executor) ParallelBatch(ctx context.Context, settings policy.Settings, calls []toolapi.Call, maximum int) (int, error) {
	var prior []parallelTarget
	var retained int64
	end := min(len(calls), maximum)
	for i, call := range calls[:end] {
		if err := ctx.Err(); err != nil {
			return i, err
		}
		effect, paths, err := e.parallelPaths(settings, call)
		if err != nil {
			return i, err
		}
		if effect == policy.Command {
			return max(1, i), nil
		}
		current := make([]parallelTarget, 0, len(paths))
		cost := int64(len(call.Arguments)) * 4
		for _, path := range paths {
			info, err := path.root.Stat(path.relative)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return i, err
			}
			if info != nil && info.Mode().IsRegular() {
				cost += info.Size() * 4
			}
			for _, previous := range prior {
				if pathContains(path.absolute, previous.absolute) || pathContains(previous.absolute, path.absolute) ||
					info != nil && previous.info != nil && os.SameFile(info, previous.info) {
					return i, nil
				}
			}
			current = append(current, parallelTarget{path, info})
		}
		if i > 0 && retained+cost > maxParallelBytes {
			return i, nil
		}
		retained += cost
		prior = append(prior, current...)
	}
	return end, nil
}

func (e *Executor) parallelPaths(settings policy.Settings, call toolapi.Call) (policy.Effect, []target, error) {
	spec, input, err := toolcatalog.Parse(call)
	if err != nil {
		return 0, nil, err
	}
	if spec.Effect == policy.Command {
		return spec.Effect, nil, nil
	}
	if spec.Name == "create_directory" {
		dirs, err := e.directoryTargets(settings, input.Paths)
		var targets []target
		for _, path := range input.Paths {
			resolved, resolveErr := e.resolve(settings.Scope, path)
			if resolveErr != nil {
				return 0, nil, resolveErr
			}
			targets = append(targets, resolved)
		}
		for _, dir := range dirs {
			if dir.info == nil {
				targets = append(targets, dir.target)
			}
		}
		return spec.Effect, targets, err
	}
	resolved, err := e.resolve(settings.Scope, input.Path)
	if err != nil {
		return 0, nil, err
	}
	return spec.Effect, []target{resolved}, nil
}
func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
