// Purpose: Widen values changed by a loop before inspecting its possible repeated effects.
// Pattern: Conservative abstract state update.
// Usage: Loop flow scans assignments and directory-changing builtins without executing iterations.
// Related: internal/policy/command/flow.go, internal/policy/command/state.go, internal/policy/command/builtins.go.
package command

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func (a *analyzer) invalidateLoop(stmts []*syntax.Stmt, s *state) {
	for _, st := range stmts {
		syntax.Walk(st, func(node syntax.Node) bool {
			switch n := node.(type) {
			case *syntax.Assign:
				if n.Name != nil {
					s.invalidate(n.Name.Value)
				}
			case *syntax.ParamExp:
				if n.Param != nil && n.Exp != nil && (n.Exp.Op == syntax.AssignUnset || n.Exp.Op == syntax.AssignUnsetOrNull) {
					s.invalidate(n.Param.Value)
				}
			case *syntax.ArithmExp:
				a.arithmeticWrites(n.X, s)
			case *syntax.CallExpr:
				if len(n.Args) == 0 {
					break
				}
				name := n.Args[0].Lit()
				switch name {
				case "cd":
					s.dir = ""
					s.invalidate("PWD")
				case ".", "source", "eval":
					s.forget()
				case "read":
					a.read(a.words(n.Args[1:], s), s)
				case "export", "readonly", "local":
					for _, arg := range n.Args[1:] {
						if name, _, ok := strings.Cut(arg.Lit(), "="); ok && syntax.ValidName(name) {
							s.invalidate(name)
						}
					}
				}
			}
			return true
		})
	}
}
