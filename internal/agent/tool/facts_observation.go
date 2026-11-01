// Purpose: Recover compact effects from committed operation facts without executing tools again.
// Pattern: Pure observation projection.
// Usage: Interrupted-work recovery applies operation records newer than its checkpoint.
// Related: internal/agent/tool/facts.go, internal/agent/tool/operation.go.
package tool

import (
	"slices"
	"sort"
	"unicode/utf8"
)

// Observe preserves known partial mutations and reports whether derived state changed.
func (f *Facts) Observe(op Operation) bool {
	command := op.Name == "execute_command" || op.Name == "wait_command" || op.Name == "cancel_command"
	changes := 0
	for _, change := range op.Changes {
		if change.BeforeDigest != change.AfterDigest || change.Destination != "" {
			changes++
		}
	}
	if len(op.Versions) == 0 && changes == 0 && !command {
		return false
	}
	if len(op.Versions) > 0 || changes > 0 {
		// Existing runtime facts are sorted. Normalize explicitly supplied facts
		// only at a real update boundary, never for a query-only result.
		less := func(i, j int) bool { return f.Files[i].Path < f.Files[j].Path }
		if !sort.SliceIsSorted(f.Files, less) {
			sort.Slice(f.Files, less)
		}
		original := len(f.Files)
		find := func(path string) int {
			return sort.Search(original, func(i int) bool { return f.Files[i].Path >= path })
		}
		updates := make(map[string]FileFact, len(op.Versions)+2*changes)
		for path, version := range op.Versions {
			changed := false
			if i := find(path); i < original && f.Files[i].Path == path {
				changed = f.Files[i].Changed
			}
			updates[path] = observedFile(op, path, version, changed)
		}
		for _, change := range op.Changes {
			if change.BeforeDigest == change.AfterDigest && change.Destination == "" {
				continue
			}
			version := change.AfterDigest
			if version == "missing" || change.Destination != "" {
				version = "missing"
			}
			if version == "" && op.Name == "create_directory" {
				version = "directory"
			}
			updates[change.Path] = observedFile(op, change.Path, version, true)
			if change.Destination != "" {
				updates[change.Destination] = observedFile(op, change.Destination, change.AfterDigest, true)
			}
		}
		for path, file := range updates {
			if i := find(path); i < original && f.Files[i].Path == path {
				f.Files[i] = file
			} else {
				f.Files = append(f.Files, file)
			}
		}
		if len(f.Files) != original {
			fresh := f.Files[original:]
			sort.Slice(fresh, func(i, j int) bool { return fresh[i].Path < fresh[j].Path })
			if original > 0 && f.Files[original-1].Path > fresh[0].Path {
				pending := slices.Clone(fresh)
				left, right, at := original-1, len(pending)-1, len(f.Files)-1
				for right >= 0 {
					if left >= 0 && f.Files[left].Path > pending[right].Path {
						f.Files[at] = f.Files[left]
						left--
					} else {
						f.Files[at] = pending[right]
						right--
					}
					at--
				}
			}
		}
		if changes > 0 {
			for i := range f.Commands {
				f.Commands[i].BeforeLaterChanges = true
			}
		}
	}
	if command {
		f.observeCommand(op)
	}
	return true
}

// reportExcerpt bounds model-facing quotations; full source records remain in session storage.
func reportExcerpt(value string) (string, bool) {
	const limit = 2048
	if len(value) <= limit {
		return value, false
	}
	end := limit
	for !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end], true
}

func observedFile(op Operation, path, version string, changed bool) FileFact {
	observation := "present"
	if version == "missing" {
		observation = "missing"
	} else if version == "" {
		observation = "unavailable"
	}
	return FileFact{Path: path, Version: version, Changed: changed, Stale: version == "", SourceID: op.CallID, ObservedRun: op.RunID, Revision: op.Revision, Observation: observation}
}
