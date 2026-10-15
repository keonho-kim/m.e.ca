// Purpose: Parse the supported Plan search command subset into read-only query values.
// Pattern: Parse boundary; no shell or external executable is invoked.
// Usage: Agent and local preparation use PlanQuery before permitting Plan command inspection.
// Related: internal/policy/shell/query_options.go, internal/tools/local/query.go.
package shell

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

const PlanQueryHelp = "Plan supports one literal query: ls [-aAl1] [path]; find [path] [-type f|d] [-name glob] [-maxdepth N]; rg --files [-g glob] [path]; rg/grep [-nriF] pattern [path]. Shell operators, expansion, redirects and executable options require leaving Plan mode."

type Query struct {
	Name, Path, Pattern, Glob                            string
	Content, Hidden, Long, Recursive, Insensitive, Fixed bool
	Directories, Files                                   bool
	MaxDepth                                             int
}

func PlanQuery(text string) (Query, error) {
	fail := func() (Query, error) {
		return Query{}, fmt.Errorf("Plan approval is required for commands outside read-only queries. %s", PlanQueryHelp)
	}
	tree, err := syntax.NewParser(syntax.Variant(syntax.LangPOSIX)).Parse(strings.NewReader(text), "")
	if err != nil || len(tree.Stmts) != 1 {
		return fail()
	}
	stmt := tree.Stmts[0]
	if stmt.Background || stmt.Negated || stmt.Coprocess || len(stmt.Redirs) > 0 {
		return fail()
	}
	call, ok := stmt.Cmd.(*syntax.CallExpr)
	if !ok || len(call.Assigns) > 0 || len(call.Args) == 0 {
		return fail()
	}
	args := make([]string, len(call.Args))
	for i, word := range call.Args {
		value, ok := literal(word.Parts)
		if !ok || strings.ContainsRune(value, 0) {
			return fail()
		}
		// Unquoted shell globs and tilde expansion are intentionally not emulated.
		for _, part := range word.Parts {
			if lit, ok := part.(*syntax.Lit); ok && strings.ContainsAny(lit.Value, "*?[]~\\") {
				return fail()
			}
		}
		args[i] = value
	}
	q := Query{Name: args[0], Path: ".", Pattern: "*", MaxDepth: -1}
	switch q.Name {
	case "ls":
		err = q.listOptions(args[1:])
	case "find":
		err = q.findOptions(args[1:])
	case "rg", "grep":
		err = q.searchOptions(args[1:])
	default:
		return fail()
	}
	if err != nil {
		return fail()
	}
	return q, nil
}
