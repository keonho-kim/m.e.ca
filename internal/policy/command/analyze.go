// Purpose: Analyze shell syntax for explicit hazardous effects without executing shell code.
// Pattern: Conservative syntax-tree analysis.
// Usage: Local command preparation and execution call Analyze with a filesystem resolver.
// Related: internal/policy/command/flow.go, internal/policy/command/words.go, internal/policy/safety.go.
package command

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/keonho-kim/v-code/internal/policy"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

type Context struct {
	Project, Home string
	Environment   []string
	Resolve       func(directory, path string, removal bool) (string, error)
}

type analyzer struct {
	ctx          context.Context
	input        Context
	result       policy.Safety
	err          error
	depth, steps int
}

type state struct {
	dir       string
	vars      map[string]expand.Variable
	unknown   map[string]bool
	functions map[string]*syntax.Stmt
	dynamic   bool
}

func Analyze(ctx context.Context, text string, input Context) (policy.Safety, error) {
	if input.Resolve == nil || !filepath.IsAbs(input.Project) {
		return policy.Safety{}, fmt.Errorf("shell analysis requires a project and target resolver")
	}
	a := &analyzer{ctx: ctx, input: input}
	s := &state{dir: input.Project, vars: map[string]expand.Variable{}, unknown: map[string]bool{}, functions: map[string]*syntax.Stmt{}}
	expand.ListEnviron(input.Environment...).Each(func(name string, value expand.Variable) bool {
		s.vars[name] = value
		return true
	})
	s.set("PWD", input.Project)
	s.parameters(nil, 0)
	s.set("0", "/bin/sh")
	for _, name := range []string{"?", "$", "!", "-", "PPID", "RANDOM", "SECONDS"} {
		s.unknown[name] = true
	}
	a.executableEnvironment(s)
	a.parse(text, syntax.LangPOSIX, s)
	return a.result, a.err
}

func (a *analyzer) executableEnvironment(s *state) {
	// Startup files and loader hooks execute code outside the supplied AST.
	for _, name := range []string{"ENV", "BASH_ENV", "SHELLOPTS", "BASHOPTS", "LD_PRELOAD", "DYLD_INSERT_LIBRARIES"} {
		if s.unknown[name] || s.vars[name].Str != "" {
			a.result.Ask("Executable environment settings require approval")
		}
	}
}

func (a *analyzer) parse(text string, lang syntax.LangVariant, s *state) {
	if !a.enter() {
		return
	}
	defer func() { a.depth-- }()
	file, err := syntax.NewParser(syntax.Variant(lang)).Parse(strings.NewReader(text), "command")
	if err != nil {
		a.err = fmt.Errorf("cannot analyze shell syntax: %w", err)
		return
	}
	a.statements(file.Stmts, s)
}

func (a *analyzer) enter() bool {
	if a.err != nil {
		return false
	}
	if err := a.ctx.Err(); err != nil {
		a.err = err
		return false
	}
	a.steps++
	// Complexity limits govern analysis only. Unresolved code needs approval,
	// never an optimistic decision after silently skipping part of the program.
	if a.depth >= 48 || a.steps > 20000 {
		a.result.Ask("Shell analysis complexity exceeded; execution needs inspection")
		return false
	}
	a.depth++
	return true
}

func (s *state) clone() *state {
	n := &state{dir: s.dir, dynamic: s.dynamic, vars: map[string]expand.Variable{}, unknown: map[string]bool{}, functions: map[string]*syntax.Stmt{}}
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
	s.dir = ""
	s.vars = map[string]expand.Variable{}
	s.functions = map[string]*syntax.Stmt{}
}

func (s *state) join(other *state) {
	if s.dir != other.dir {
		s.dir = ""
	}
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
