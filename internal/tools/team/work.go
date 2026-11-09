// Purpose: Resolve contribution targets and guard actual file effects against current team work.
// Pattern: Filesystem adapter for room-owned contribution decisions.
// Usage: Declaration, preparation and execution use the same canonical path rules.
// Related: internal/tools/team/admission.go, internal/tools/team/local.go, internal/agent/team/work.go.
package team

import (
	"path/filepath"
	"slices"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
)

func (e *Executor) canonicalPaths(paths []string, declaration bool) ([]string, error) {
	out := make([]string, 0, len(paths))
	root := ""
	if declaration && len(paths) > 0 {
		var err error
		root, err = filesystem.Canonical(e.Project)
		if err != nil {
			return nil, err
		}
	}
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(e.Project, path)
		}
		path, err := filesystem.Canonical(path)
		if err != nil {
			return nil, err
		}
		if declaration {
			relative, err := filepath.Rel(path, root)
			if err == nil && (relative == "." || filepath.IsLocal(relative)) {
				return nil, faults.Errorf(diagcatalog.TeamClaimConcreteFilesOrSubdirectoriesOwningProject)
			}
		}
		out = append(out, path)
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}
