// Purpose: Unwrap supported executable wrappers and inspect nested shell programs and input scripts.
// Pattern: Recursive syntax dispatch.
// Usage: Command analysis resolves wrappers before effect classification.
// Related: internal/policy/command/calls.go, internal/policy/command/find.go.
package command

import (
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

func (a *analyzer) wrapper(name string, args []word, redirs []*syntax.Redirect, s *state) bool {
	switch name {
	case "sh", "dash", "bash", "zsh", "ksh", "ash":
		a.shell(name, args, redirs, s)
		return true
	case "env":
		work := s.clone()
		for len(args) > 0 {
			arg := args[0]
			if !arg.known {
				a.result.Ask("Unresolved env wrapper requires approval")
				return true
			}
			if arg.text == "--" {
				args = args[1:]
				break
			}
			if arg.text == "-i" || arg.text == "--ignore-environment" {
				work.vars = map[string]expand.Variable{}
				work.unknown = map[string]bool{}
				work.dynamic = false
				args = args[1:]
				continue
			}
			if arg.text == "-u" || arg.text == "--unset" {
				if len(args) < 2 || !args[1].known {
					a.result.Ask("Unresolved env unset requires approval")
					return true
				}
				delete(work.vars, args[1].text)
				args = args[2:]
				continue
			}
			if arg.text == "-C" || arg.text == "--chdir" {
				if len(args) < 2 {
					a.result.Ask("Incomplete env directory option requires inspection")
					return true
				}
				a.cd(args[1:2], work)
				args = args[2:]
				continue
			}
			if strings.HasPrefix(arg.text, "--chdir=") {
				a.cd([]word{{text: strings.TrimPrefix(arg.text, "--chdir="), known: true}}, work)
				args = args[1:]
				continue
			}
			if strings.HasPrefix(arg.text, "--unset=") {
				delete(work.vars, strings.TrimPrefix(arg.text, "--unset="))
				args = args[1:]
				continue
			}
			if key, value, ok := strings.Cut(arg.text, "="); ok && !strings.HasPrefix(key, "-") {
				work.set(key, value)
				args = args[1:]
				continue
			}
			if strings.HasPrefix(arg.text, "-") {
				a.result.Ask("Unsupported env option requires approval")
				return true
			}
			break
		}
		a.invoke(args, redirs, work)
		return true
	case "command":
		if len(args) > 0 && args[0].known && (args[0].text == "-v" || args[0].text == "-V") {
			return true
		}
		for len(args) > 0 && args[0].known && (args[0].text == "-p" || args[0].text == "--") {
			args = args[1:]
		}
		work := s.clone()
		work.functions = map[string]*syntax.Stmt{}
		a.invoke(args, redirs, work)
		return true
	case "exec", "builtin", "nohup", "setsid", "nice", "timeout", "stdbuf", "time":
		for len(args) > 0 && args[0].known && strings.HasPrefix(args[0].text, "-") {
			option := args[0].text
			args = args[1:]
			if option == "--" {
				break
			}
			takes := name == "nice" && (option == "-n" || option == "--adjustment") || name == "timeout" && (option == "-k" || option == "--kill-after" || option == "-s" || option == "--signal") || name == "time" && (option == "-o" || option == "-f") || name == "exec" && option == "-a" || name == "stdbuf" && (option == "-o" || option == "-i" || option == "-e")
			if takes && len(args) > 0 {
				args = args[1:]
				continue
			}
			if strings.Contains(option, "=") || name == "nice" && len(option) > 1 && option[1] >= '0' && option[1] <= '9' || name == "time" && option == "-p" || name == "nohup" || name == "setsid" {
				continue
			}
			a.result.Ask("Wrapper option requires inspection")
		}
		if name == "timeout" && len(args) > 0 {
			args = args[1:]
		}
		a.invoke(args, redirs, s)
		return true
	case "busybox", "toybox":
		a.invoke(args, redirs, s)
		return true
	case "xargs":
		a.xargs(args, s)
		return true
	}
	return false
}

func (a *analyzer) shell(name string, args []word, redirs []*syntax.Redirect, s *state) {
	lang := syntax.LangPOSIX
	if name == "bash" {
		lang = syntax.LangBash
	} else if name == "zsh" {
		lang = syntax.LangZsh
	} else if name == "ksh" {
		lang = syntax.LangMirBSDKorn
	}
	work := s.clone()
	for i, arg := range args {
		if !arg.known {
			a.result.Ask("Dynamic nested shell requires approval")
			return
		}
		if strings.HasPrefix(arg.text, "-") && !strings.HasPrefix(arg.text, "--") && strings.Contains(arg.text, "c") {
			if i+1 >= len(args) || !args[i+1].known {
				a.result.Ask("Dynamic shell program requires approval")
				return
			}
			work.parameters(args[i+2:], 0)
			a.parse(args[i+1].text, lang, work)
			return
		}
		if !strings.HasPrefix(arg.text, "-") {
			a.result.Ask("External shell script requires approval")
			return
		}
	}
	for _, r := range redirs {
		if r.Hdoc != nil {
			body := a.word(r.Hdoc, s, true)[0]
			if body.known {
				a.parse(body.text, lang, work)
			} else {
				a.result.Ask("Dynamic shell input requires approval")
			}
			return
		}
	}
	a.result.Ask("Shell program supplied at runtime requires approval")
}

func (a *analyzer) xargs(args []word, s *state) {
	for len(args) > 0 && args[0].known && strings.HasPrefix(args[0].text, "-") {
		flag := args[0].text
		args = args[1:]
		if flag == "--" {
			break
		}
		if flag == "-I" || flag == "-n" || flag == "-P" || flag == "-L" || flag == "-d" || flag == "-a" || flag == "--arg-file" {
			if len(args) > 0 {
				args = args[1:]
			}
		} else if flag != "-0" && flag != "-r" && flag != "-t" {
			a.result.Ask("xargs option requires inspection")
		}
	}
	if len(args) == 0 {
		return
	}
	// Literal arguments still reveal forbidden targets; stdin contributes
	// unknown additional arguments, so effects cannot be automatically approved.
	a.invoke(append(args, word{}), nil, s.clone())
	if args[0].known && filepath.Base(args[0].text) != "echo" && filepath.Base(args[0].text) != "printf" {
		a.result.Ask("xargs receives runtime arguments; inspect before execution")
	}
}
