// Purpose: Preserve a bounded set of known working directories across shell control-flow merges.
// Pattern: Conservative abstract state operations.
// Usage: State joins retain candidates; cd updates directory variables and widening forgets them.
// Related: internal/policy/command/state.go, internal/policy/command/builtins.go, internal/policy/command/targets.go.
package command

import "slices"

// An empty directory set means unknown, not an unreachable branch.
// Beyond this analysis budget, directory-dependent mutations require approval.
const maxDirectoryCandidates = 32

func mergeDirectories(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	out := slices.Clone(left)
	for _, directory := range right {
		if slices.Contains(out, directory) {
			continue
		}
		if len(out) == maxDirectoryCandidates {
			return nil
		}
		out = append(out, directory)
	}
	return out
}

func (s *state) changeDirectories(next []string) {
	if len(s.directories) == 1 {
		s.set("OLDPWD", s.directories[0])
	} else {
		s.invalidate("OLDPWD")
	}
	s.directories = next
	if len(next) == 1 {
		s.set("PWD", next[0])
	} else {
		s.invalidate("PWD")
	}
}

func (s *state) forgetDirectory() {
	s.directories = nil
	s.invalidate("PWD")
	s.invalidate("OLDPWD")
}
