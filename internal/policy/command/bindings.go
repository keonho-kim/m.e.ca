// Purpose: Read shared initial values and branch-local overrides without copying the environment.
// Pattern: Immutable base with copy-on-write overlays.
// Usage: State owns all writes; expansion adapters read value and each.
// Related: internal/policy/command/state.go, internal/policy/command/words.go.
package command

import (
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

type initialValues struct{ vars map[string]expand.Variable }
type bindings struct {
	base      *initialValues
	vars      map[string]expand.Variable
	unknown   map[string]bool
	functions map[string]*syntax.Stmt
}

func (s *state) value(name string) (expand.Variable, bool) {
	if value, ok := s.vars[name]; ok {
		return value, true
	}
	if s.base != nil {
		value, ok := s.base.vars[name]
		return value, ok
	}
	return expand.Variable{}, false
}
func (s *state) each(fn func(string, expand.Variable) bool) {
	if s.base != nil {
		for name, value := range s.base.vars {
			if _, changed := s.vars[name]; !changed && !fn(name, value) {
				return
			}
		}
	}
	for name, value := range s.vars {
		if !fn(name, value) {
			return
		}
	}
}
