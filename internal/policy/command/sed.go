// Purpose: Separate sed invocation options, programs and in-place targets before applying shared path policy.
// Pattern: Command-specific argument parser.
// Usage: Shell effect inspection dispatches sed and gsed invocations here.
// Related: internal/policy/command/sed_program.go, internal/policy/command/effects.go.
package command

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func (a *analyzer) sed(args []word, s *state) {
	inPlace, program, options := false, false, true
	backup := ""
	var files, programs []word
	takeProgram := func(value word, external bool) {
		program = true
		if external {
			a.ask("External sed script requires approval")
		} else {
			programs = append(programs, value)
		}
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			if !program || options {
				a.ask("Dynamic sed program or target requires approval")
			}
			if !program {
				program = true
				programs = append(programs, arg)
			} else {
				files = append(files, arg)
			}
			continue
		}
		if options && arg.text == "--" {
			options = false
			continue
		}
		if options && strings.HasPrefix(arg.text, "--") {
			option, value, attached := strings.Cut(arg.text, "=")
			switch option {
			case "--expression", "--file":
				source := word{text: value, known: true}
				if !attached {
					if i+1 >= len(args) {
						a.ask("Unrecognized sed program requires approval")
						continue
					}
					i++
					source = args[i]
				}
				takeProgram(source, option == "--file")
			case "--in-place":
				inPlace = true
				backup = value
			case "--quiet", "--silent", "--regexp-extended", "--unbuffered", "--separate", "--null-data", "--sandbox", "--posix", "--debug", "--follow-symlinks":
			case "--help", "--version":
				return
			default:
				a.ask("Unrecognized sed program requires approval")
			}
			continue
		}
		if options && strings.HasPrefix(arg.text, "-") && arg.text != "-" {
			for j := 1; j < len(arg.text); j++ {
				flag := arg.text[j]
				switch flag {
				case 'i':
					inPlace = true
					backup = arg.text[j+1:]
					// BSD sed uses a separate empty backup suffix; GNU accepts an attached suffix.
					if j == len(arg.text)-1 && i+1 < len(args) && args[i+1].known && args[i+1].text == "" {
						i++
					}
					j = len(arg.text)
				case 'e', 'f':
					source := word{text: arg.text[j+1:], known: true}
					if source.text == "" {
						if i+1 >= len(args) {
							a.ask("Unrecognized sed program requires approval")
							break
						}
						i++
						source = args[i]
					}
					takeProgram(source, flag == 'f')
					j = len(arg.text)
				case 'n', 'E', 'r', 'u', 's', 'z':
				default:
					a.ask("Unrecognized sed program requires approval")
				}
			}
			continue
		}
		if !program {
			takeProgram(arg, false)
		} else {
			files = append(files, arg)
		}
	}
	for _, source := range programs {
		if !source.known {
			a.ask("Dynamic sed program or target requires approval")
			continue
		}
		effects := sedProgram(source.text)
		if effects.unknown {
			a.ask("Unrecognized sed program requires approval")
		}
		if effects.dynamic {
			a.ask("Sed executes runtime-generated commands; approval required")
		}
		for _, path := range effects.writes {
			a.target(word{text: path, known: true}, s, false)
		}
		for _, code := range effects.commands {
			a.parse(code, syntax.LangPOSIX, s.clone())
		}
	}
	if inPlace {
		for _, file := range files {
			a.target(file, s, false)
			if backup != "" {
				if !file.known {
					a.target(word{}, s, false)
					continue
				}
				path := file.text + backup
				if strings.Contains(backup, "*") {
					path = strings.ReplaceAll(backup, "*", file.text)
				}
				a.target(word{text: path, known: true}, s, false)
			}
		}
	}
}
