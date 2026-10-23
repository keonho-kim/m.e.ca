// Purpose: Inspect move sources and actual destination entries before applying project-boundary policy.
// Pattern: Command-specific argument and filesystem analysis.
// Usage: The mv effect classifier passes arguments and possible working directories here.
// Related: internal/policy/command/effects.go, internal/policy/command/targets.go, internal/tools/local/command_safety.go.
package command

import (
	"path/filepath"
	"slices"
	"strings"
)

type moveOperands struct {
	sources                             []word
	destination                         *word
	targetDirectory, noTarget, noFollow bool
}

func (a *analyzer) move(args []word, s *state) {
	args = slices.Clone(args)
	for i := range args {
		if args[i].glob {
			args[i].known = false
		}
	}
	if len(args) == 1 && args[0].known && (args[0].text == "--help" || args[0].text == "--version") {
		return
	}
	move := a.moveOperands(args)
	for _, source := range move.sources {
		// A single unresolved source can expand to multiple operands at runtime.
		a.target(source, s, true)
		if a.err != nil {
			return
		}
	}
	if move.destination == nil {
		return
	}
	if !move.destination.known {
		a.target(*move.destination, s, true)
		return
	}
	bases := s.directories
	if filepath.IsAbs(move.destination.text) {
		bases = []string{""}
	}
	if len(bases) == 0 {
		a.ask("Mutation working directory is unresolved")
		return
	}
	for _, base := range bases {
		if !a.enter() {
			return
		}
		destination, err := a.input.Resolve(base, move.destination.text, true)
		a.depth--
		if err != nil {
			a.ask("Mutation target could not be canonicalized")
			continue
		}
		if destination.Symlink && !move.noTarget && !move.noFollow {
			if !a.enter() {
				return
			}
			followed, err := a.input.Resolve(base, move.destination.text, false)
			a.depth--
			if err != nil {
				a.ask("Mutation target could not be canonicalized")
				continue
			}
			if followed.Directory {
				destination = followed
			}
		}
		if move.targetDirectory || destination.Directory && !move.noTarget {
			for _, source := range move.sources {
				if !source.known {
					a.directoryTarget(word{text: destination.Path, known: true}, s, false)
					continue
				}
				path := filepath.Join(destination.Path, filepath.Base(source.text))
				a.target(word{text: path, known: true}, s, true)
			}
		} else {
			a.target(word{text: destination.Path, known: true}, s, true)
		}
		if a.err != nil {
			return
		}
	}
}

func (a *analyzer) moveOperands(args []word) moveOperands {
	var move moveOperands
	var values []word
	options, seenOperand := true, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if options && arg.known && arg.text == "--" {
			options = false
			continue
		}
		if !options || !arg.known || !strings.HasPrefix(arg.text, "-") || arg.text == "-" {
			values = append(values, arg)
			seenOperand = true
			continue
		}
		if seenOperand {
			a.ask("Move options require inspection")
		}
		switch {
		case arg.text == "--target-directory":
			move.targetDirectory = true
			if i+1 < len(args) {
				i++
				move.destination = &args[i]
			}
		case strings.HasPrefix(arg.text, "--target-directory="):
			move.targetDirectory = true
			move.destination = &word{text: strings.TrimPrefix(arg.text, "--target-directory="), known: true}
		case arg.text == "--no-target-directory":
			move.noTarget = true
		case arg.text == "--force", arg.text == "--interactive", arg.text == "--no-clobber", arg.text == "--verbose", arg.text == "--update", strings.HasPrefix(arg.text, "--update="), arg.text == "--no-copy", arg.text == "--debug":
		case strings.HasPrefix(arg.text, "--"):
			a.ask("Move options require inspection")
		default:
			for j := 1; j < len(arg.text); j++ {
				switch arg.text[j] {
				case 'f', 'i', 'n', 'v', 'u':
				case 'h':
					move.noFollow = true
				case 'T':
					move.noTarget = true
				case 't':
					move.targetDirectory = true
					if j+1 < len(arg.text) {
						move.destination = &word{text: arg.text[j+1:], known: true}
					} else if i+1 < len(args) {
						i++
						move.destination = &args[i]
					}
					j = len(arg.text)
				default:
					a.ask("Move options require inspection")
				}
			}
		}
	}
	move.sources = values
	if !move.targetDirectory && len(values) >= 2 {
		move.sources, move.destination = values[:len(values)-1], &values[len(values)-1]
	}
	return move
}
