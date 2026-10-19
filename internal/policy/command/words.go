// Purpose: Expand statically knowable shell words while inspecting executable substitutions separately.
// Pattern: Static expansion adapter.
// Usage: Command and redirect analysis consume known words or explicit unknowns.
// Related: internal/policy/command/calls.go, internal/policy/command/flow.go.
package command

import (
	"errors"
	"io/fs"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

type word struct {
	text        string
	known, glob bool
}
type environment struct {
	state      *state
	unresolved bool
}

func (e *environment) Get(name string) expand.Variable {
	if strings.HasPrefix(name, "HOME ") {
		e.unresolved = true
		return expand.Variable{Set: true, Kind: expand.String}
	}
	value, found := e.state.vars[name]
	if e.state.unknown[name] || e.state.dynamic && !found {
		e.unresolved = true
	}
	return value
}
func (e *environment) Each(fn func(string, expand.Variable) bool) {
	for name, value := range e.state.vars {
		if !fn(name, value) {
			return
		}
	}
}

func (a *analyzer) substitutions(w *syntax.Word, s *state) bool {
	unknown := false
	syntax.Walk(w, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.CmdSubst:
			a.statements(n.Stmts, s.clone())
			unknown = true
			return false
		case *syntax.ProcSubst:
			a.statements(n.Stmts, s.clone())
			unknown = true
			return false
		case *syntax.ArithmExp:
			a.result.Ask("Arithmetic expansion with runtime state requires approval")
			s.forget()
			unknown = true
			return false
		case *syntax.ParamExp:
			if n.Excl || n.NestedParam != nil || n.Index != nil || n.Exp != nil && (n.Exp.Op == syntax.AssignUnset || n.Exp.Op == syntax.AssignUnsetOrNull) {
				unknown = true
				a.result.Ask("Indirect or state-changing parameter expansion requires approval")
				s.forget()
			}
		}
		return true
	})
	return unknown
}

func (a *analyzer) word(w *syntax.Word, s *state, literal bool) []word {
	if w == nil {
		return nil
	}
	unknown := a.substitutions(w, s)
	if unknown {
		return []word{{}}
	}
	// Named-user tilde expansion may invoke os/user.Lookup in the expansion
	// library. Leave it unresolved rather than introduce implicit host I/O.
	if len(w.Parts) > 0 {
		if lit, ok := w.Parts[0].(*syntax.Lit); ok && strings.HasPrefix(lit.Value, "~") && lit.Value != "~" && !strings.HasPrefix(lit.Value, "~/") {
			return []word{{}}
		}
	}
	env := &environment{state: s}
	glob := false
	cfg := &expand.Config{Env: env, ReadDir2: func(string) ([]fs.DirEntry, error) { glob = true; return nil, errors.New("glob expansion deferred") }}
	if literal {
		value, err := expand.Literal(cfg, w)
		return []word{{text: value, known: err == nil && !unknown && !env.unresolved}}
	}
	copy := *w
	if syntax.SplitBraces(&copy) {
		// Fields performs brace enumeration even for POSIX input. Defer it:
		// large ranges/products must never allocate proportional expansions.
		return []word{{}}
	}
	fields, err := expand.Fields(cfg, w)
	if err != nil || unknown || env.unresolved {
		value, _ := expand.Literal(&expand.Config{Env: env}, w)
		return []word{{text: value, glob: glob}}
	}
	out := make([]word, 0, len(fields))
	for _, field := range fields {
		out = append(out, word{text: field, known: true})
	}
	return out
}

func (a *analyzer) words(words []*syntax.Word, s *state) []word {
	var out []word
	for _, w := range words {
		out = append(out, a.word(w, s, false)...)
	}
	return out
}

func (a *analyzer) assign(v *syntax.Assign, s *state) {
	if v.Name == nil || v.Naked || v.Index != nil || v.Array != nil || v.Append {
		a.result.Ask("Dynamic shell assignment requires approval")
		s.forget()
		return
	}
	value := word{known: true}
	if v.Value != nil {
		value = a.word(v.Value, s, true)[0]
	}
	if value.known {
		s.set(v.Name.Value, value.text)
	} else {
		s.unknown[v.Name.Value] = true
	}
}
