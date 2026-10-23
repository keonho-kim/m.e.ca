// Purpose: Update only the shell state affected by builtins used in compound commands.
// Pattern: Explicit abstract state transitions.
// Usage: Command dispatch handles cd, set and read through these methods.
// Related: internal/policy/command/calls.go, internal/policy/command/state.go.
package command

import (
	"path/filepath"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func (a *analyzer) set(args []word, s *state) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			s.invalidateParameters()
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
		s.invalidateParameters()
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
				s.changeDirectories(nil)
				return
			}
			path = arg
		}
	}
	if path.text == "-" {
		path = word{text: s.vars["OLDPWD"].Str, known: !s.unknown["OLDPWD"] && s.vars["OLDPWD"].IsSet()}
	}
	if !path.known || !filepath.IsAbs(path.text) && (len(s.directories) == 0 || s.vars["CDPATH"].Str != "" || s.unknown["CDPATH"]) {
		s.changeDirectories(nil)
		return
	}
	bases := s.directories
	if filepath.IsAbs(path.text) {
		bases = []string{""}
	}
	var next []string
	for _, base := range bases {
		if !a.enter() {
			s.changeDirectories(nil)
			return
		}
		resolved, err := a.input.Resolve(base, path.text, false)
		a.depth--
		if err != nil {
			s.changeDirectories(nil)
			return
		}
		if !slices.Contains(next, resolved) {
			next = append(next, resolved)
		}
	}
	s.changeDirectories(next)
}

func (a *analyzer) read(args []word, s *state) {
	names := 0
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			s.invalidateVariables()
			return
		}
		if strings.HasPrefix(arg.text, "-") {
			if arg.text == "-p" || arg.text == "-u" || arg.text == "-t" || arg.text == "-n" || arg.text == "-N" || arg.text == "-d" {
				i++
			}
			continue
		}
		if syntax.ValidName(arg.text) {
			s.invalidate(arg.text)
			names++
		}
	}
	if names == 0 {
		s.invalidate("REPLY")
	}
}

func specialBuiltin(name string) bool {
	switch name {
	case ":", ".", "break", "continue", "eval", "exec", "exit", "export", "readonly", "return", "set", "shift", "times", "trap", "unset":
		return true
	}
	return false
}
