// Purpose: Reconcile file-content observations against freshly checked filesystem versions.
// Pattern: Pure state reconciliation.
// Usage: The run owner supplies scoped versions after a command; unknown outcomes stay stale.
// Related: internal/agent/completion/state.go, internal/agent/run/file_facts.go.
package completion

import "slices"

func (s State) FileTargets() []string {
	paths := map[string]bool{}
	for _, f := range s.CurrentFiles {
		paths[f.Path] = true
	}
	for _, r := range s.Receipts {
		if r.Kind == "command" {
			continue
		}
		for path, version := range r.Versions {
			if version != "" {
				paths[path] = true
			}
		}
	}
	out := make([]string, 0, len(paths))
	for path := range paths {
		out = append(out, path)
	}
	slices.Sort(out)
	return out
}

func (s *State) ReconcileFiles(current map[string]string) {
	receipts := slices.Clone(s.Receipts)
	changed := false
	for i, r := range receipts {
		if r.Kind == "command" || len(r.Versions) == 0 {
			continue
		}
		matches := true
		for path, version := range r.Versions {
			if version == "" || current[path] != version {
				matches = false
				break
			}
		}
		receipts[i].Stale = !matches
		changed = changed || !matches && !r.Stale
	}
	if changed {
		for i := range receipts {
			if receipts[i].Kind == "command" {
				receipts[i].BeforeLaterChanges = true
			}
		}
	}
	s.Receipts = receipts
	s.reconcileCurrent(current)
}
