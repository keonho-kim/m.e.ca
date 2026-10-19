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
	before := s.clone()
	if len(c.Args) == 0 {
		for _, v := range c.Assigns {
			a.assign(v, s)
		}
		return s.clone()
	}
	args := a.words(c.Args, s)
	work := s
	if len(c.Assigns) > 0 {
		work = s.clone()
		for _, v := range c.Assigns {
			a.assign(v, work)
		}
	}
	a.invoke(args, redirs, work)
	if work != s {
		s.join(before)
	}
	if len(args) > 0 && args[0].known && (args[0].text == "export" || args[0].text == "readonly" || args[0].text == "unset" || args[0].text == "set") {
		return s.clone()
	}
	return before
}

func (a *analyzer) invoke(args []word, redirs []*syntax.Redirect, s *state) {
	if len(args) == 0 || !a.enter() {
		return
	}
	defer func() { a.depth-- }()
	a.executableEnvironment(s)
	if !args[0].known {
		a.result.Ask("Unresolved command name requires approval")
		s.forget()
		return
	}
	called := args[0].text
	name := filepath.Base(called)
	args = args[1:]
	if body, ok := s.functions[name]; ok && !strings.Contains(called, "/") {
		work := s.clone()
		work.parameters(args, 1)
		a.statements([]*syntax.Stmt{body}, work)
		*s = *work
		return
	}
	if a.wrapper(name, args, redirs, s) {
		return
	}
	switch name {
	case "sudo", "su", "doas", "pkexec":
		a.err = fmt.Errorf("operation blocked: privilege escalation via %s", name)
	case "cd":
		a.cd(args, s)
	case "eval":
		var text []string
		for _, arg := range args {
			if !arg.known {
				a.result.Ask("Dynamic eval requires approval")
				s.forget()
				return
			}
			text = append(text, arg.text)
		}
		a.parse(strings.Join(text, " "), syntax.LangPOSIX, s)
	case ".", "source":
		a.result.Ask("External shell script sourcing requires approval")
		s.forget()
	case "unset":
		for _, arg := range args {
			if arg.known && !strings.HasPrefix(arg.text, "-") {
				delete(s.vars, arg.text)
				delete(s.unknown, arg.text)
				delete(s.functions, arg.text)
			} else {
				s.forget()
			}
		}
	case "export", "readonly", "local":
		for _, arg := range args {
			key, value, ok := strings.Cut(arg.text, "=")
			if arg.known && ok {
				s.set(key, value)
			} else if !arg.known {
				s.forget()
				a.result.Ask("Dynamic environment change requires approval")
			}
		}
	case "set":
		a.set(args, s)
	case "read", "getopts", "shift", "alias", "unalias":
		a.result.Ask("Runtime shell state changes require approval")
		s.forget()
	case "trap":
		if len(args) > 0 && args[0].known {
			a.parse(args[0].text, syntax.LangPOSIX, s.clone())
		} else {
			a.result.Ask("Dynamic shell trap requires approval")
		}
	default:
		a.effects(name, args, s)
	}
}

func (a *analyzer) set(args []word, s *state) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			a.result.Ask("Dynamic shell options require approval")
			s.forget()
			return
		}
		if arg.text == "--" {
			s.parameters(args[i+1:], 1)
			return
		}
		if arg.text == "-o" || arg.text == "+o" {
			if i+1 < len(args) && args[i+1].known && (args[i+1].text == "errexit" || args[i+1].text == "nounset" || args[i+1].text == "pipefail" || args[i+1].text == "xtrace" || args[i+1].text == "noglob") {
				i++
				continue
			}
		}
		if len(arg.text) > 1 && (arg.text[0] == '-' || arg.text[0] == '+') && strings.Trim(arg.text[1:], "euxfnvC") == "" {
			continue
		}
		a.result.Ask("Unsupported shell options require approval")
		s.forget()
		return
	}
}

func (a *analyzer) cd(args []word, s *state) {
	var path word
	if len(args) == 0 {
		path = word{text: s.vars["HOME"].Str, known: !s.unknown["HOME"] && s.vars["HOME"].IsSet()}
	} else {
		for _, arg := range args {
			if arg.known && (arg.text == "--" || arg.text == "-P" || arg.text == "-L") {
				continue
			}
			if path.known || path.text != "" {
				s.dir = ""
				return
			}
			path = arg
		}
	}
	if path.text == "-" {
		path = word{text: s.vars["OLDPWD"].Str, known: !s.unknown["OLDPWD"] && s.vars["OLDPWD"].IsSet()}
	}
	if !path.known || !filepath.IsAbs(path.text) && (s.dir == "" || s.vars["CDPATH"].Str != "" || s.unknown["CDPATH"]) {
		s.dir = ""
		s.unknown["PWD"] = true
		return
	}
	resolved, err := a.input.Resolve(s.dir, path.text, false)
	if err != nil {
		s.dir = ""
		s.unknown["PWD"] = true
		return
	}
	s.set("OLDPWD", s.dir)
	s.dir = resolved
	s.set("PWD", resolved)
}
