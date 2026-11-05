// Purpose: Reuse one parsed shell program and frozen environment across fresh safety observations.
// Pattern: Immutable prepared analysis input.
// Usage: Local command preparation retains Program until its one execution finishes.
// Related: internal/policy/command/analyze.go, internal/tools/local/command_operation.go.
package command

import (
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
	"strings"
)

type Program struct {
	file *syntax.File
	base *initialValues
}

func Parse(text string, environment []string) (*Program, error) {
	file, err := syntax.NewParser(syntax.Variant(syntax.LangPOSIX)).Parse(strings.NewReader(text), "command")
	if err != nil {
		return nil, &SyntaxError{Cause: err}
	}
	base := &initialValues{vars: make(map[string]expand.Variable, len(environment))}
	expand.ListEnviron(environment...).Each(func(name string, value expand.Variable) bool { base.vars[name] = value; return true })
	return &Program{file: file, base: base}, nil
}
