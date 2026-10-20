// Purpose: Classify shell calls and maintain literal variables, directories and function parameters.
// Pattern: Static command dispatch.
// Usage: Flow analysis passes command nodes and redirects here.
// Related: internal/policy/command/wrappers.go, internal/policy/command/effects.go.
package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func (a *analyzer) call(c *syntax.CallExpr, redirs []*syntax.Redirect, s *state) *state {
	if len(c.Args) == 0 {
		for _, v := range c.Assigns {
			a.assign(v, s)
		}
		return s.clone()
	}
	args := a.word(c.Args[0], s, false)
	inspect := len(args) == 0 || !args[0].known
	if len(args) > 0 && args[0].known {
		name := filepath.Base(args[0].text)
		_, function := s.functions[name]
		inspect = inspectionRequired(name) || function
	}
	for _, arg := range c.Args[1:] {
		if inspect {
			args = append(args, a.word(arg, s, false)...)
		} else {
			a.substitutions(arg, s)
		}
	}
	failure := s.clone()
	work := s
	if len(c.Assigns) > 0 {
		work = s.clone()
		for _, v := range c.Assigns {
			a.assign(v, work)
		}
	}
	a.invoke(args, redirs, work, true)
	if work != s {
		if len(args) == 0 || !args[0].known || !specialBuiltin(args[0].text) {
			for _, assignment := range c.Assigns {
				if assignment.Name == nil {
					continue
				}
				name := assignment.Name.Value
				delete(work.vars, name)
				delete(work.unknown, name)
				if value, ok := failure.vars[name]; ok {
					work.vars[name] = value
				}
				if failure.unknown[name] {
					work.unknown[name] = true
				}
			}
		}
		*s = *work
	}
	if len(args) > 0 && args[0].known && (args[0].text == "export" || args[0].text == "readonly" || args[0].text == "unset" || args[0].text == "set") {
		return s.clone()
	}
	return failure
}

func (a *analyzer) invoke(args []word, redirs []*syntax.Redirect, s *state, functions bool) {
	if len(args) == 0 || !a.enter() {
		return
	}
	defer func() { a.depth-- }()
	a.executableEnvironment(s)
	if !args[0].known {
		defer a.finding("")()
		a.ask("Unresolved command name requires approval")
		s.forget()
		return
	}
	called := args[0].text
	name := filepath.Base(called)
	args = args[1:]
	if body, ok := s.functions[name]; ok && functions && !strings.Contains(called, "/") {
		work := s.clone()
		work.parameters(args, 1)
		a.statements([]*syntax.Stmt{body}, work)
		*s = *work
		return
	}
	if !inspectionRequired(name) {
		return
	}
	defer a.finding(name)()
	if a.wrapper(name, args, redirs, s) {
		return
	}
	switch name {
	case "sudo", "su", "doas", "pkexec":
		a.err = &BlockedError{Cause: fmt.Errorf("operation blocked: privilege escalation via %s", name)}
	case "cd":
		a.cd(args, s)
	case "eval":
		var text []string
		for _, arg := range args {
			if !arg.known {
				a.ask("Dynamic eval requires approval")
				s.forget()
				return
			}
			text = append(text, arg.text)
		}
		a.parse(strings.Join(text, " "), syntax.LangPOSIX, s)
	case ".", "source":
		a.ask("External shell script sourcing requires approval")
		s.forget()
	case "unset":
		for _, arg := range args {
			if arg.known && !strings.HasPrefix(arg.text, "-") {
				delete(s.vars, arg.text)
				delete(s.unknown, arg.text)
				delete(s.functions, arg.text)
			} else {
				s.invalidateVariables()
			}
		}
	case "export", "readonly", "local":
		for _, arg := range args {
			key, value, ok := strings.Cut(arg.text, "=")
			if arg.known && ok {
				s.set(key, value)
			} else if !arg.known {
				if syntax.ValidName(key) {
					s.invalidate(key)
				} else {
					s.invalidateVariables()
				}
			}
		}
	case "set":
		a.set(args, s)
	case "read":
		a.read(args, s)
	case "getopts":
		if len(args) > 1 && args[1].known {
			s.invalidate(args[1].text)
		} else {
			s.invalidateVariables()
		}
		s.invalidate("OPTARG")
		s.invalidate("OPTIND")
	case "shift":
		s.invalidateParameters()
	case "alias", "unalias":
		a.ask("Command alias changes require approval")
		s.forget()
	case "trap":
		if len(args) > 0 && args[0].known {
			a.parse(args[0].text, syntax.LangPOSIX, s.clone())
		} else {
			a.ask("Dynamic shell trap requires approval")
		}
	default:
		a.effects(name, args, s)
	}
}
