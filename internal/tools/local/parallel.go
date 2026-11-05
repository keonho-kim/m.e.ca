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
const preparationBytes int64 = 1024 * 1024

type parallelTarget struct {
	target
	info os.FileInfo
}

func (e *Executor) ParallelBatch(ctx context.Context, settings policy.Settings, calls []toolapi.Call, maximum int) (result toolapi.BatchPlan, err error) {
	var largestTemporary int64
	prepareFiles := true
	plan := func(count int, err error) (toolapi.BatchPlan, error) {
		workers := 1
		if prepareFiles && largestTemporary > 0 {
			workers = max(1, min(4, count, int(preparationBytes/largestTemporary)))
		}
		return toolapi.BatchPlan{Count: count, PreparationConcurrency: workers}, err
	}
	if len(calls) == 0 {
		return plan(0, nil)
	}
	if commandControl(calls[0].Name) {
		_, _, err := toolcatalog.Parse(calls[0])
		return plan(1, err)
	}
	// Serialize image decoding and keep compressed results from accumulating in a
	// wide wave. File changes and running shell commands retain their own limits.
	if len(calls) > 0 && calls[0].Name == "view_image" {
		return plan(1, nil)
	}
	if len(calls) > 0 && calls[0].Name == "execute_command" {
		end := 0
		var retained int64
		for _, call := range calls[:min(len(calls), maximum, max(1, e.commands.Available()))] {
			if call.Name != "execute_command" {
				break
			}
			if _, _, err := toolcatalog.Parse(call); err != nil {
				return plan(end, err)
			}
			cost := int64(len(call.Arguments)) * 4
			if end > 0 && retained+cost > maxParallelBytes {
				break
			}
			retained += cost
			end++
		}
		return plan(end, nil)
	}
	var prior []parallelTarget
	var retained int64
	if err := e.boundary.Check(); err != nil {
		return plan(0, err)
	}
	defer func() { err = errors.Join(err, e.boundary.Check()) }()
	end := min(len(calls), maximum)
	for i, call := range calls[:end] {
		if call.Name == "view_image" {
			return plan(max(1, i), nil)
		}
		if err := ctx.Err(); err != nil {
			return plan(i, err)
		}
		effect, paths, err := e.parallelPaths(settings, call)
		if err != nil {
			return plan(i, err)
		}
		if effect == policy.Command {
			return plan(max(1, i), nil)
		}
		current := make([]parallelTarget, 0, len(paths))
		cost := int64(len(call.Arguments)) * 4
		temporary := cost
		for _, path := range paths {
			info, err := path.root.Stat(path.relative)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return plan(i, err)
			}
			if info != nil && info.Mode().IsRegular() {
				cost += info.Size() * 4
				temporary += info.Size() * 2
			}
			for _, previous := range prior {
				if pathContains(path.absolute, previous.absolute) || pathContains(previous.absolute, path.absolute) ||
					info != nil && previous.info != nil && os.SameFile(info, previous.info) {
					return plan(i, nil)
				}
			}
			current = append(current, parallelTarget{path, info})
		}
		if i > 0 && retained+cost > maxParallelBytes {
			return plan(i, nil)
		}
		retained += cost
		largestTemporary = max(largestTemporary, temporary)
		prepareFiles = prepareFiles && (call.Name == "write_file" || call.Name == "edit_file")
		prior = append(prior, current...)
	}
	return plan(end, nil)
}

func (e *Executor) parallelPaths(settings policy.Settings, call toolapi.Call) (policy.Effect, []target, error) {
	spec, input, err := toolcatalog.Parse(call)
	if err != nil {
		return 0, nil, err
	}
	if spec.Effect == policy.Command {
		return spec.Effect, nil, nil
	}
	if commandControl(spec.Name) {
		return policy.Command, nil, nil
	}
	if spec.Name == "create_directory" {
		dirs, requested, err := e.collectDirectories(settings, input.Paths)
		targets := requested
		for _, dir := range dirs {
			if dir.info == nil {
				targets = append(targets, dir.target)
			}
		}
		return spec.Effect, targets, err
	}
	resolved, err := e.resolveTarget(settings.Scope, input.Path)
	if err != nil {
		return 0, nil, err
	}
	return spec.Effect, []target{resolved}, nil
}
func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
