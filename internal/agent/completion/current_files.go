// Purpose: Project current known filesystem observations without rewriting historical outcomes.
// Pattern: Pure state projection.
// Usage: Observe records known effects; ReconcileFiles supplies fresh scoped probes.
// Related: internal/agent/tool/facts.go, internal/agent/completion/freshness.go.
package completion

import (
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"slices"
)

func (s *State) observeCurrent(r Receipt, changed []string, unknown bool) {
	files := map[string]tool.FileFact{}
	for _, f := range s.CurrentFiles {
		for _, path := range changed {
			if overlaps(path, f.Path) {
				f.Stale = true
			}
		}
		f.Stale = f.Stale || unknown
		files[f.Path] = f
	}
	for path, version := range r.Versions {
		if version == "" {
			continue
		}
		prior := files[path]
		changedPath := r.Changed
		if r.Kind == "command" {
			changedPath = slices.Contains(changed, path)
		}
		files[path] = tool.FileFact{Path: path, Version: version, Changed: prior.Changed || changedPath, Stale: r.Stale, SourceID: r.ID, ObservedRun: r.ObservedRun, Revision: r.Revision, Observation: observationKind(version)}
	}
	s.setCurrent(files)
}

func observationKind(version string) string {
	if version == "missing" {
		return "missing"
	}
	if version == "" {
		return "unavailable"
	}
	return "present"
}

func (s *State) reconcileCurrent(current map[string]string) {
	// Current observations include partial effects from failed operations.
	files := map[string]tool.FileFact{}
	for _, f := range s.Facts().Files {
		files[f.Path] = f
	}
	for _, r := range s.Receipts {
		for path, version := range r.Versions {
			if _, ok := files[path]; !ok {
				files[path] = tool.FileFact{Path: path, Version: version, Changed: r.Changed, SourceID: r.ID, ObservedRun: r.ObservedRun, Revision: r.Revision}
			}
		}
	}
	for path, f := range files {
		version := current[path]
		f.Observation = observationKind(version)
		f.Stale = version == ""
		if version != "" {
			f.Version = version
		}
		files[path] = f
	}
	s.setCurrent(files)
}

func (s *State) setCurrent(files map[string]tool.FileFact) {
	s.CurrentFiles = nil
	for _, f := range files {
		s.CurrentFiles = append(s.CurrentFiles, f)
	}
	slices.SortFunc(s.CurrentFiles, func(a, b tool.FileFact) int {
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	})
}
