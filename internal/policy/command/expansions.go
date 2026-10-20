// Purpose: Isolate expansion dependencies and state writes without treating unknown data as executable code.
// Pattern: AST inspection helpers.
// Usage: Word analysis distinguishes literal words, parameter assignments and arithmetic writes.
// Related: internal/policy/command/words.go, internal/policy/command/state.go.
package command

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func literalWord(w *syntax.Word) bool {
	literal := true
	syntax.Walk(w, func(node syntax.Node) bool {
		switch node.(type) {
		case nil, *syntax.Word, *syntax.Lit, *syntax.SglQuoted, *syntax.DblQuoted:
			return true
		default:
			literal = false
			return false
		}
	})
	return literal
}

func assignmentPrefix(w *syntax.Word) string {
	if len(w.Parts) > 0 {
		if part, ok := w.Parts[0].(*syntax.Lit); ok {
			if name, _, found := strings.Cut(part.Value, "="); found && syntax.ValidName(name) {
				return name + "="
			}
		}
	}
	return ""
}

func (a *analyzer) arithmeticWrites(expr syntax.ArithmExpr, s *state) {
	syntax.Walk(expr, func(node syntax.Node) bool {
		var target syntax.ArithmExpr
		switch n := node.(type) {
		case *syntax.UnaryArithm:
			if n.Op == syntax.Inc || n.Op == syntax.Dec {
				target = n.X
			}
		case *syntax.BinaryArithm:
			switch n.Op {
			case syntax.Assgn, syntax.AddAssgn, syntax.SubAssgn, syntax.MulAssgn, syntax.QuoAssgn, syntax.RemAssgn, syntax.AndAssgn, syntax.OrAssgn, syntax.XorAssgn, syntax.ShlAssgn, syntax.ShrAssgn:
				target = n.X
			}
		}
		if word, ok := target.(*syntax.Word); ok {
			if name := word.Lit(); syntax.ValidName(name) {
				s.invalidate(name)
			}
		}
		return true
	})
}

func (a *analyzer) parameterAssignment(n *syntax.ParamExp, s *state) {
	if n.Param == nil {
		a.substitutions(n.Exp.Word, s)
		s.invalidateVariables()
		return
	}
	name := n.Param.Value
	prior, found := s.vars[name]
	known := !s.unknown[name] && (!s.dynamic || found)
	assign := !prior.IsSet() || n.Exp.Op == syntax.AssignUnsetOrNull && prior.Str == ""
	if known && !assign {
		return
	}
	values := a.word(n.Exp.Word, s, true)
	if known && len(values) == 1 && values[0].known {
		s.set(name, values[0].text)
	} else {
		s.invalidate(name)
	}
}

// Unknown quoted expansions still occupy one argv field, except positional/array @.
func oneField(w *syntax.Word, literal bool) bool {
	if literal {
		return true
	}
	var partSingle func(syntax.WordPart, bool) bool
	partSingle = func(part syntax.WordPart, quoted bool) bool {
		switch n := part.(type) {
		case *syntax.Lit:
			return quoted || !strings.ContainsAny(n.Value, "*?[")
		case *syntax.SglQuoted:
			return true
		case *syntax.DblQuoted:
			for _, part := range n.Parts {
				if !partSingle(part, true) {
					return false
				}
			}
			return true
		case *syntax.ParamExp:
			return quoted && n.Param != nil && n.Param.Value != "@" && n.Index == nil
		case *syntax.CmdSubst, *syntax.ArithmExp:
			return quoted
		}
		return false
	}
	for _, part := range w.Parts {
		if !partSingle(part, false) {
			return false
		}
	}
	return true
}
