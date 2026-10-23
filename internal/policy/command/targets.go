// Purpose: Check every known working-directory candidate for mutation paths and directory bounds.
// Pattern: Filesystem-policy boundary.
// Usage: Registered effect inspectors share target and directoryTarget before approval.
// Related: internal/policy/safety.go, internal/policy/command/effects.go, internal/tools/local/command_safety.go.
package command

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/keonho-kim/v-code/internal/policy"
)

func (a *analyzer) target(value word, s *state, removal bool) {
	if !value.known {
		a.ask("Mutation target cannot be resolved statically")
		if value.glob {
			// Check a known directory prefix without expanding or scanning it.
			prefix := value.text
			if i := strings.IndexAny(prefix, "*?["); i >= 0 {
				prefix = prefix[:i]
			}
			if !strings.HasSuffix(prefix, string(filepath.Separator)) {
				prefix = filepath.Dir(prefix)
			}
			a.directoryTarget(word{text: prefix, known: true}, s, removal)
		}
		return
	}
	if value.text == "" {
		return
	}
	paths := a.resolveTargets(value.text, s, removal)
	if a.err != nil {
		return
	}
	for _, path := range paths {
		decision, err := policy.Mutation(a.input.Project, a.input.Home, path, removal)
		if err != nil {
			a.err = &BlockedError{Cause: err}
			return
		}
		a.merge(decision)
	}
}

func (a *analyzer) directoryTarget(value word, s *state, removal bool) {
	if !value.known {
		a.ask("Mutation working directory is unresolved")
		return
	}
	directories := a.resolveTargets(value.text, s, false)
	if a.err != nil {
		return
	}
	for _, directory := range directories {
		decision, err := policy.MutationWithin(a.input.Project, a.input.Home, directory, removal)
		if err != nil {
			a.err = &BlockedError{Cause: err}
			return
		}
		a.merge(decision)
	}
}

func (a *analyzer) resolveTargets(value string, s *state, removal bool) []string {
	bases := s.directories
	if filepath.IsAbs(value) {
		bases = []string{""}
	}
	if len(bases) == 0 {
		a.ask("Mutation working directory is unresolved")
		return nil
	}
	var paths []string
	for _, base := range bases {
		if !a.enter() {
			return paths
		}
		path, err := a.input.Resolve(base, value, removal)
		a.depth--
		if err != nil {
			a.ask("Mutation target could not be canonicalized")
			continue
		}
		if !slices.Contains(paths, path) {
			paths = append(paths, path)
		}
	}
	return paths
}
