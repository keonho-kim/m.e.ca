// Purpose: Analyze shell syntax for explicit hazardous effects without executing shell code.
// Pattern: Conservative syntax-tree analysis.
// Usage: Prepared programs analyze fresh filesystem observations without executing shell code.
// Related: internal/policy/command/flow.go, internal/policy/command/words.go, internal/policy/safety.go.
package command

import (
	"context"

	"path/filepath"
	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/policy"
	"mvdan.cc/sh/v3/syntax"

	// ResolvedPath describes the entry observed after the requested path resolution.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type ResolvedPath struct {
	Path               string
	Directory, Symlink bool
}

type Context struct {
	Project, Home string
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

func (p *Program) Analyze(ctx context.Context, input Context) (Report, error) {
	if input.Resolve == nil || !filepath.IsAbs(input.Project) {
		return Report{}, faults.Errorf(diagcatalog.CommandShellAnalysisRequiresAProjectAndTarget)
	}
	a := &analyzer{ctx: ctx, input: input}
	s := &state{directories: []string{input.Project}, bindings: &bindings{base: p.base}}
	s.set("PWD", input.Project)
	s.parameters(nil, 0)
	s.set("0", "/bin/sh")
	for _, name := range []string{"?", "$", "!", "-", "PPID", "RANDOM", "SECONDS"} {
		s.markUnknown(name)
	}
	a.executableEnvironment(s)
	if a.enter() {
		a.statements(p.file.Stmts, s)
		a.depth--
	}
	return Report{Safety: a.result, Findings: a.findings}, a.err
}

func (a *analyzer) executableEnvironment(s *state) {
	// Startup files and loader hooks execute code outside the supplied AST.
	for _, name := range []string{"ENV", "BASH_ENV", "SHELLOPTS", "BASHOPTS", "LD_PRELOAD", "DYLD_INSERT_LIBRARIES"} {
		value, _ := s.value(name)
		if s.unknown[name] || value.Str != "" {
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
