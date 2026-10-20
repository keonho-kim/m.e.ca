// Purpose: Preserve xargs runtime data as unknown operands while inspecting only its actual child command.
// Pattern: Wrapper argument parser.
// Usage: Shell wrapper dispatch unwraps xargs before inspecting registered child effects.
// Related: internal/policy/command/wrappers.go, internal/policy/command/inspection.go.
package command

import "strings"

func (a *analyzer) xargs(args []word, s *state) {
	replacement := ""
	for len(args) > 0 && args[0].known && strings.HasPrefix(args[0].text, "-") {
		flag := args[0].text
		args = args[1:]
		if flag == "--" {
			break
		}
		if flag == "-I" || flag == "--replace" {
			if flag == "--replace" {
				replacement = "{}"
				continue
			}
			if len(args) == 0 || !args[0].known {
				a.ask("xargs option requires inspection")
				return
			}
			replacement = args[0].text
			args = args[1:]
			continue
		}
		if strings.HasPrefix(flag, "-I") && len(flag) > 2 {
			replacement = flag[2:]
			continue
		}
		if strings.HasPrefix(flag, "--replace=") {
			replacement = strings.TrimPrefix(flag, "--replace=")
			continue
		}
		option, _, attached := strings.Cut(flag, "=")
		switch option {
		case "-n", "-P", "-L", "-d", "-a", "-E", "-s", "--max-args", "--max-procs", "--max-lines", "--delimiter", "--arg-file", "--eof", "--max-chars":
			if !attached && len(args) > 0 {
				args = args[1:]
			}
		case "-0", "-r", "-t", "-x", "-p", "--null", "--no-run-if-empty", "--verbose", "--exit", "--interactive":
		default:
			if len(flag) > 2 && strings.ContainsRune("nPLdsE", rune(flag[1])) {
				continue
			}
			a.ask("xargs option requires inspection")
		}
	}
	if len(args) == 0 {
		return
	}
	nested := append([]word(nil), args...)
	if replacement != "" {
		for i, arg := range nested {
			if !arg.known || strings.Contains(arg.text, replacement) {
				nested[i] = word{}
			}
		}
	} else {
		nested = append(nested, word{})
	}
	// A runtime operand only needs resolution when a registered child operation uses it.
	a.invoke(nested, nil, s.clone(), true)
}
