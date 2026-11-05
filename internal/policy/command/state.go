// Purpose: Share immutable shell environments and isolate only changed bindings across branches.
// Pattern: Copy-on-write abstract shell state.
// Usage: Command flow clones state cheaply; every binding mutation passes through writable.
// Related: internal/policy/command/bindings.go, internal/policy/command/calls.go, internal/policy/command/words.go.
package command

import (
	"maps"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
	"slices"
	"strconv"
)

type state struct {
	directories []string
	*bindings
	shared  bool
	dynamic bool
}

func (s *state) clone() *state {
	s.shared = true
	n := *s
	return &n
}

func (s *state) writable() {
	if s.bindings == nil {
		s.bindings = &bindings{}
	}
	if !s.shared {
		return
	}
	n := *s.bindings
	n.vars, n.unknown, n.functions = maps.Clone(n.vars), maps.Clone(n.unknown), maps.Clone(n.functions)
	s.bindings, s.shared = &n, false
}

func (s *state) put(name string, value expand.Variable) {
	s.writable()
	if s.vars == nil {
		s.vars = make(map[string]expand.Variable)
	}
	s.vars[name] = value
	delete(s.unknown, name)
}
func (s *state) set(name, value string) {
	s.put(name, expand.Variable{Set: true, Kind: expand.String, Str: value})
}
func (s *state) unset(name string) { s.put(name, expand.Variable{}) }
func (s *state) forget() {
	s.dynamic, s.directories = true, nil
	s.bindings = &bindings{unknown: maps.Clone(s.unknown)}
	s.shared = false
}
func (s *state) function(name string, body *syntax.Stmt) {
	if body == nil && s.functions[name] == nil {
		return
	}
	s.writable()
	if s.functions == nil {
		s.functions = make(map[string]*syntax.Stmt)
	}
	if body == nil {
		delete(s.functions, name)
	} else {
		s.functions[name] = body
	}
}

func (s *state) join(other *state) {
	s.directories = mergeDirectories(s.directories, other.directories)
	s.dynamic = s.dynamic || other.dynamic
	if s.bindings == other.bindings {
		return
	}
	// Shared environment entries cannot differ: compare only branch overlays.
	if s.base == other.base {
		for name := range s.vars {
			s.joinVariable(other, name)
		}
		for name := range other.vars {
			s.joinVariable(other, name)
		}
	} else {
		s.each(func(name string, _ expand.Variable) bool { s.joinVariable(other, name); return true })
		other.each(func(name string, _ expand.Variable) bool { s.joinVariable(other, name); return true })
	}
	for name, unknown := range other.unknown {
		if unknown {
			s.invalidate(name)
		}
	}
	for name, body := range s.functions {
		if other.functions[name] != body {
			s.function(name, nil)
		}
	}
}
func (s *state) joinVariable(other *state, name string) {
	if s.unknown[name] {
		return
	}
	left, _ := s.value(name)
	right, _ := other.value(name)
	if other.unknown[name] || !sameVariable(left, right) {
		s.invalidate(name)
	}
}

func positional(name string) bool {
	if name == "" || name == "0" {
		return false
	}
	for _, c := range name {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
func (s *state) parameters(args []word, offset int) {
	s.each(func(name string, _ expand.Variable) bool {
		if positional(name) {
			s.unset(name)
		}
		return true
	})
	s.writable()
	for name := range s.unknown {
		if positional(name) {
			delete(s.unknown, name)
		}
	}
	var values []string
	known := true
	for i, arg := range args {
		name := strconv.Itoa(i + offset)
		if arg.known {
			s.set(name, arg.text)
		} else {
			s.invalidate(name)
			known = false
		}
		if i+offset > 0 {
			values = append(values, arg.text)
		}
	}
	s.set("#", strconv.Itoa(len(values)))
	for _, name := range []string{"@", "*"} {
		s.put(name, expand.Variable{Set: true, Kind: expand.Indexed, List: values})
		if !known {
			s.markUnknown(name)
		}
	}
}
func (s *state) markUnknown(name string) {
	if s.unknown[name] {
		return
	}
	s.writable()
	if s.unknown == nil {
		s.unknown = make(map[string]bool)
	}
	s.unknown[name] = true
}
func (s *state) invalidate(name string) {
	s.unset(name)
	s.markUnknown(name)
}
func (s *state) invalidateVariables() {
	s.each(func(name string, _ expand.Variable) bool { s.invalidate(name); return true })
	s.dynamic = true
}
func (s *state) invalidateParameters() {
	s.each(func(name string, _ expand.Variable) bool {
		if positional(name) {
			s.invalidate(name)
		}
		return true
	})
	for _, name := range []string{"@", "*", "#"} {
		s.invalidate(name)
	}
}
func sameVariable(a, b expand.Variable) bool {
	return a.Set == b.Set && a.Local == b.Local && a.Exported == b.Exported && a.ReadOnly == b.ReadOnly && a.Kind == b.Kind && a.Str == b.Str && slices.Equal(a.List, b.List) && slices.Equal(a.Indexes, b.Indexes) && maps.Equal(a.Map, b.Map)
}
