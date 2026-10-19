// Purpose: Track possible shell state across sequences, branches, loops and nested command bodies.
// Pattern: Conservative control-flow analysis.
// Usage: The analyzer visits statements without running them.
// Related: internal/policy/command/analyze.go, internal/policy/command/calls.go.
package command

import "mvdan.cc/sh/v3/syntax"

func (a *analyzer) statements(stmts []*syntax.Stmt, s *state) {
	for _, st := range stmts {
		failure := a.statement(st, s)
		s.join(failure)
		if a.err != nil {
			return
		}
	}
}

// s is the successful continuation; the return value is the failed continuation.
// This keeps `cd dir && rm file` distinct from `cd dir; rm file`.
func (a *analyzer) statement(st *syntax.Stmt, s *state) *state {
	before := s.clone()
	if st == nil || !a.enter() {
		return before
	}
	defer func() { a.depth-- }()
	for _, r := range st.Redirs {
		a.redirect(r, s)
	}
	failure := s.clone()
	switch c := st.Cmd.(type) {
	case *syntax.CallExpr:
		failure = a.call(c, st.Redirs, s)
	case *syntax.Block:
		a.statements(c.Stmts, s)
		failure = s.clone()
	case *syntax.Subshell:
		a.statements(c.Stmts, s.clone())
	case *syntax.BinaryCmd:
		xfail := a.statement(c.X, s)
		switch c.Op {
		case syntax.AndStmt:
			yfail := a.statement(c.Y, s)
			xfail.join(yfail)
			failure = xfail
		case syntax.OrStmt:
			yfail := a.statement(c.Y, xfail)
			s.join(xfail)
			failure = yfail
		default:
			a.statement(c.Y, before.clone())
			*s = *before.clone()
			failure = before.clone()
		}
	case *syntax.IfClause:
		a.statements(c.Cond, s)
		other := s.clone()
		a.statements(c.Then, s)
		if c.Else != nil {
			a.statement(&syntax.Stmt{Cmd: c.Else}, other)
		}
		s.join(other)
		failure = s.clone()
	case *syntax.ForClause:
		loop, ok := c.Loop.(*syntax.WordIter)
		if !ok {
			a.result.Ask("Dynamic shell loop requires approval")
			s.forget()
			a.statements(c.Do, s)
			break
		}
		values := a.words(loop.Items, s)
		if len(values) == 0 {
			values = []word{{}}
		}
		for _, value := range values {
			body := s.clone()
			if value.known {
				body.set(loop.Name.Value, value.text)
			} else {
				body.unknown[loop.Name.Value] = true
			}
			a.statements(c.Do, body)
			before.join(body)
		}
		*s = *before
		failure = s.clone()
	case *syntax.WhileClause:
		a.result.Ask("Shell loop with runtime-dependent iterations requires approval")
		a.statements(c.Cond, s)
		s.forget()
		a.statements(c.Do, s)
		failure = s.clone()
	case *syntax.CaseClause:
		a.words([]*syntax.Word{c.Word}, s)
		for _, item := range c.Items {
			body := s.clone()
			a.words(item.Patterns, body)
			a.statements(item.Stmts, body)
			before.join(body)
		}
		*s = *before
		failure = s.clone()
	case *syntax.FuncDecl:
		if c.Name == nil {
			a.result.Ask("Anonymous or multi-name shell function requires inspection")
			a.statement(c.Body, s.clone())
		} else {
			s.functions[c.Name.Value] = c.Body
		}
		failure = s.clone()
	case *syntax.TimeClause:
		failure = a.statement(c.Stmt, s)
	case *syntax.DeclClause:
		for _, assignment := range c.Args {
			a.assign(assignment, s)
		}
		failure = s.clone()
	case nil:
	default:
		a.result.Ask("Unsupported shell state changes require approval")
		s.forget()
	}
	if st.Negated {
		success := s.clone()
		*s = *failure
		failure = success
	}
	if st.Background || st.Coprocess || st.Disown {
		s.join(before)
		failure.join(before)
	}
	return failure
}
