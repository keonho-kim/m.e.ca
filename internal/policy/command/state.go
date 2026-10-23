// Purpose: Track independently known shell values, directory candidates and function definitions.
// Pattern: Abstract shell state.
// Usage: Command flow and word analysis copy and merge state without executing code.
// Related: internal/policy/command/directories.go, internal/policy/command/calls.go, internal/policy/command/words.go.
package command

import (
	"reflect"
	"slices"
	"strconv"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

type state struct {
	directories []string
	vars        map[string]expand.Variable
	unknown     map[string]bool
	functions   map[string]*syntax.Stmt
	dynamic     bool
}

func (s *state) clone() *state {
	n := &state{directories: slices.Clone(s.directories), dynamic: s.dynamic, vars: map[string]expand.Variable{}, unknown: map[string]bool{}, functions: map[string]*syntax.Stmt{}}
	for k, v := range s.vars {
		n.vars[k] = v
	}
	for k, v := range s.unknown {
		n.unknown[k] = v
	}
	for k, v := range s.functions {
		n.functions[k] = v
	}
	return n
}

func (s *state) set(name, value string) {
	s.vars[name] = expand.Variable{Set: true, Kind: expand.String, Str: value}
	delete(s.unknown, name)
}

func (s *state) forget() {
	s.dynamic = true
	s.directories = nil
	s.vars = map[string]expand.Variable{}
	s.functions = map[string]*syntax.Stmt{}
}

func (s *state) join(other *state) {
	s.directories = mergeDirectories(s.directories, other.directories)
	if s.dynamic != other.dynamic {
		s.dynamic = true
	}
	for k, v := range s.vars {
		if other.unknown[k] || !reflect.DeepEqual(v, other.vars[k]) {
			s.unknown[k] = true
			delete(s.vars, k)
		}
	}
	for k := range other.vars {
		if _, ok := s.vars[k]; !ok {
			s.unknown[k] = true
		}
	}
	for k, v := range other.unknown {
		if v {
			s.unknown[k] = true
		}
	}
	for k, v := range s.functions {
		if other.functions[k] != v {
			delete(s.functions, k)
		}
	}
}

func (s *state) parameters(args []word, offset int) {
	for key := range s.vars {
		if _, err := strconv.Atoi(key); err == nil && key != "0" {
			delete(s.vars, key)
		}
	}
	for key := range s.unknown {
		if _, err := strconv.Atoi(key); err == nil && key != "0" {
			delete(s.unknown, key)
		}
	}
	var values []string
	known := true
	for i, arg := range args {
		key := strconv.Itoa(i + offset)
		if arg.known {
			s.set(key, arg.text)
		} else {
			s.unknown[key] = true
			known = false
		}
		if i+offset > 0 {
			values = append(values, arg.text)
		}
	}
	s.set("#", strconv.Itoa(len(values)))
	for _, name := range []string{"@", "*"} {
		s.vars[name] = expand.Variable{Set: true, Kind: expand.Indexed, List: values}
		if !known {
			s.unknown[name] = true
		} else {
			delete(s.unknown, name)
		}
	}
}

func (s *state) invalidate(name string) {
	delete(s.vars, name)
	s.unknown[name] = true
}

func (s *state) invalidateVariables() {
	for name := range s.vars {
		s.invalidate(name)
	}
	s.dynamic = true
}

func (s *state) invalidateParameters() {
	for name := range s.vars {
		if _, err := strconv.Atoi(name); err == nil && name != "0" {
			s.invalidate(name)
		}
	}
	for _, name := range []string{"@", "*", "#"} {
		s.invalidate(name)
	}
}
