// Purpose: Analyze shell syntax for explicit hazardous effects without executing shell code.
// Pattern: Conservative syntax-tree analysis.
// Usage: Local command preparation and execution call Analyze with a filesystem resolver.
// Related: internal/policy/command/flow.go, internal/policy/command/words.go, internal/policy/safety.go.
package command

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/keonho-kim/meca/internal/policy"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

// ResolvedPath describes the entry observed after the requested path resolution.
type ResolvedPath struct {
	Path               string
	Directory, Symlink bool
}

type Context struct {
	Project, Home string
	Environment   []string
	Resolve       func(directory, path string, removal bool) (ResolvedPath, error)
}

type analyzer struct {
	ctx          context.Context
	input        Context
	result       policy.Safety
	findings     []Finding
	current      *Finding
	err          error
	depth, steps int
}

func Analyze(ctx context.Context, text string, input Context) (Report, error) {
	if input.Resolve == nil || !filepath.IsAbs(input.Project) {
		return Report{}, fmt.Errorf("shell analysis requires a project and target resolver")
	}
	a := &analyzer{ctx: ctx, input: input}
	s := &state{directories: []string{input.Project}, vars: map[string]expand.Variable{}, unknown: map[string]bool{}, functions: map[string]*syntax.Stmt{}}
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
	return Report{Safety: a.result, Findings: a.findings}, a.err
}

func (a *analyzer) executableEnvironment(s *state) {
	// Startup files and loader hooks execute code outside the supplied AST.
	for _, name := range []string{"ENV", "BASH_ENV", "SHELLOPTS", "BASHOPTS", "LD_PRELOAD", "DYLD_INSERT_LIBRARIES"} {
		if s.unknown[name] || s.vars[name].Str != "" {
			a.ask("Executable environment settings require approval")
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
		a.err = &SyntaxError{Cause: err}
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
		a.ask("Shell analysis complexity exceeded; execution needs inspection")
		return false
	}
	a.depth++
	return true
}
