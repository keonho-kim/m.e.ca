// Purpose: Distinguish sed programs and option values from explicit in-place edit targets.
// Pattern: Command-specific argument parsing.
// Usage: Shell effect analysis inspects sed invocations here.
// Related: internal/policy/command/effects.go.
package command

import "strings"

func (a *analyzer) sed(args []word, s *state) {
	inPlace, program, options := false, false, true
	var files []word
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			a.result.Ask("Dynamic sed program or target requires approval")
			files = append(files, arg)
			continue
		}
		if options && arg.text == "--" {
			options = false
			continue
		}
		if options && strings.HasPrefix(arg.text, "-") && arg.text != "-" {
			switch {
			case arg.text == "-e" || arg.text == "--expression" || arg.text == "-f" || arg.text == "--file":
				i++
				program = true
			case strings.HasPrefix(arg.text, "--expression=") || strings.HasPrefix(arg.text, "--file=") || strings.HasPrefix(arg.text, "-e") || strings.HasPrefix(arg.text, "-f"):
				program = true
			case arg.text == "--in-place" || strings.HasPrefix(arg.text, "--in-place=") || !strings.HasPrefix(arg.text, "--") && strings.Contains(arg.text[1:], "i"):
				inPlace = true
			}
			continue
		}
		if !program {
			program = true
			continue
		}
		files = append(files, arg)
	}
	if inPlace {
		a.result.Ask("In-place sed editing requires approval")
		for _, file := range files {
			a.target(file, s, false)
		}
	}
}
