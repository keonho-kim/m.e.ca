// Purpose: Resolve concrete mutation paths or known directory bounds without inventing displayed filenames.
// Pattern: Filesystem-policy boundary.
// Usage: Registered effect inspectors share target and directoryTarget before approval.
// Related: internal/policy/safety.go, internal/policy/command/effects.go, internal/tools/local/command_safety.go.
package command

import (
	"path/filepath"
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
	if !filepath.IsAbs(value.text) && s.dir == "" {
		a.ask("Mutation working directory is unresolved")
		return
	}
	path, err := a.input.Resolve(s.dir, value.text, removal)
	if err != nil {
		a.ask("Mutation target could not be canonicalized")
		return
	}
	decision, err := policy.Mutation(a.input.Project, a.input.Home, path, removal)
	if err != nil {
		a.err = &BlockedError{Cause: err}
		return
	}
	a.merge(decision)
}

func (a *analyzer) directoryTarget(value word, s *state, removal bool) {
	if !value.known || !filepath.IsAbs(value.text) && s.dir == "" {
		a.ask("Mutation working directory is unresolved")
		return
	}
	directory, err := a.input.Resolve(s.dir, value.text, false)
	if err != nil {
		a.ask("Mutation target could not be canonicalized")
		return
	}
	decision, err := policy.MutationWithin(a.input.Project, a.input.Home, directory, removal)
	if err != nil {
		a.err = &BlockedError{Cause: err}
		return
	}
	a.merge(decision)
}
