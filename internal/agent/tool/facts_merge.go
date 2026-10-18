// Purpose: Combine final observations while retaining changed-file provenance and current versions.
// Pattern: Pure value merge.
// Usage: Run reports aggregate observed file and command facts.
// Related: internal/agent/tool/facts.go, internal/agent/completion/facts.go.
package tool

import "sort"

func (f *Facts) Merge(next Facts) {
	files := map[string]FileFact{}
	for _, file := range f.Files {
		files[file.Path] = file
	}
	for _, file := range next.Files {
		prior, exists := files[file.Path]
		file.Changed = file.Changed || prior.Changed
		if exists && !prior.Stale && file.Stale {
			prior.Changed = file.Changed
			files[file.Path] = prior
		} else {
			files[file.Path] = file
		}
	}
	f.Files = nil
	for _, file := range files {
		f.Files = append(f.Files, file)
	}
	sort.Slice(f.Files, func(i, j int) bool { return f.Files[i].Path < f.Files[j].Path })
	type identity struct {
		run, call string
		revision  int
	}
	indices := map[identity]int{}
	for i, c := range f.Commands {
		if c.ObservedRun != "" && c.CallID != "" {
			indices[identity{c.ObservedRun, c.CallID, c.Revision}] = i
		}
	}
	for _, c := range next.Commands {
		key := identity{c.ObservedRun, c.CallID, c.Revision}
		if index, exists := indices[key]; exists && key.run != "" && key.call != "" {
			c.BeforeLaterChanges = c.BeforeLaterChanges || f.Commands[index].BeforeLaterChanges
			f.Commands[index] = c
			continue
		}
		if key.run != "" && key.call != "" {
			indices[key] = len(f.Commands)
		}
		f.Commands = append(f.Commands, c)
	}
}
