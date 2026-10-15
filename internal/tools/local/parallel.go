// Purpose: Determine whether two local operations have independent resolved filesystem targets.
// Pattern: Filesystem scheduling adapter; execution still rechecks policy and versions.
// Usage: The runtime asks CanParallel before grouping prepared operations.
// Related: internal/tools/local/paths.go, internal/agent/run/batch.go.
package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (e *Executor) CanParallel(ctx context.Context, settings policy.Settings, left, right toolapi.Call) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	a, pa, err := e.parallelPaths(settings, left)
	if err != nil {
		return false, err
	}
	b, pb, err := e.parallelPaths(settings, right)
	if err != nil {
		return false, err
	}
	if a == policy.Command || b == policy.Command {
		return false, nil
	}
	if a == policy.Read && b == policy.Read {
		return true, nil
	}
	for _, x := range pa {
		for _, y := range pb {
			if pathContains(x.absolute, y.absolute) || pathContains(y.absolute, x.absolute) {
				return false, nil
			}
			xi, xe := x.root.Stat(x.relative)
			yi, ye := y.root.Stat(y.relative)
			if xe != nil && !errors.Is(xe, os.ErrNotExist) {
				return false, xe
			}
			if ye != nil && !errors.Is(ye, os.ErrNotExist) {
				return false, ye
			}
			if xe == nil && ye == nil && os.SameFile(xi, yi) {
				return false, nil
			}
		}
	}
	return true, nil
}
func (e *Executor) parallelPaths(settings policy.Settings, call toolapi.Call) (policy.Effect, []target, error) {
	spec, input, err := parseArguments(call)
	if err != nil {
		return 0, nil, err
	}
	if spec.effect == policy.Command {
		return spec.effect, nil, nil
	}
	if spec.name == "create_directory" {
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
		return spec.effect, targets, err
	}
	paths := []string{input.Path}
	if spec.name == "move_file" {
		paths = append(paths, input.Destination)
	}
	var targets []target
	for _, path := range paths {
		resolved, err := e.resolve(settings.Scope, path)
		if err != nil {
			return 0, nil, err
		}
		targets = append(targets, resolved)
	}
	return spec.effect, targets, nil
}
func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
