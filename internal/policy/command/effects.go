// Purpose: Recognize explicit filesystem and destructive Git effects of common shell commands.
// Pattern: Command effect classification.
// Usage: Resolved calls feed operands into the shared mutation policy.
// Related: internal/policy/command/find.go, internal/policy/command/move.go, internal/policy/command/removal.go, internal/policy/command/redirects.go, internal/policy/safety.go.
package command

import "strings"

func (a *analyzer) effects(name string, args []word, s *state) {
	switch name {
	case "rm", "rmdir", "unlink":
		parents := name == "rmdir" && removalParents(args)
		for _, arg := range a.operands(args, nil) {
			a.target(arg, s, true)
			if parents {
				a.removalParents(arg, s)
			}
		}
	case "mv":
		a.move(args, s)
	case "cp", "install", "ln":
		values := a.operands(args, []string{"-t", "--target-directory", "-S", "--suffix", "-m", "--mode", "-o", "--owner", "-g", "--group"})
		for i, arg := range args {
			if arg.known && (arg.text == "-t" || arg.text == "--target-directory") && i+1 < len(args) {
				a.target(args[i+1], s, false)
			}
			if arg.known && strings.HasPrefix(arg.text, "--target-directory=") {
				a.target(word{text: strings.TrimPrefix(arg.text, "--target-directory="), known: true}, s, false)
			}
		}
		if len(values) > 0 {
			a.target(values[len(values)-1], s, false)
		}
	case "mkdir", "touch", "truncate", "tee":
		for _, arg := range a.operands(args, []string{"-m", "--mode", "-t", "-d", "--date", "-r", "--reference", "-s", "--size"}) {
			a.target(arg, s, false)
		}
	case "chmod", "chown", "chgrp":
		simpleExecute := name == "chmod" && len(args) > 1 && args[0].known && (args[0].text == "+x" || args[0].text == "u+x" || args[0].text == "a+x")
		if !simpleExecute {
			a.ask("Permission or ownership changes require approval")
		}
		values := a.operands(args, []string{"--reference"})
		reference := false
		for _, arg := range args {
			if arg.known && (arg.text == "--reference" || strings.HasPrefix(arg.text, "--reference=")) {
				reference = true
			}
		}
		if len(values) > 0 && !reference {
			values = values[1:]
		}
		for _, arg := range values {
			a.target(arg, s, false)
		}
	case "find":
		a.find(args, s)
	case "git":
		a.git(args, s)
	case "sed", "gsed":
		a.sed(args, s)
	case "dd", "mkfs", "mke2fs", "fdisk", "sfdisk", "parted", "wipefs", "mount", "umount", "chroot":
		a.ask("System-level command requires explicit inspection")
		for _, arg := range args {
			if arg.known && strings.HasPrefix(arg.text, "of=") {
				a.target(word{text: strings.TrimPrefix(arg.text, "of="), known: true}, s, false)
			}
		}
	}
}

func (a *analyzer) operands(args []word, takesValue []string) []word {
	var values []word
	options := true
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			values = append(values, arg)
			continue
		}
		if options && arg.text == "--" {
			options = false
			continue
		}
		if options && strings.HasPrefix(arg.text, "-") && arg.text != "-" {
			for _, option := range takesValue {
				if arg.text == option {
					i++
					break
				}
			}
			continue
		}
		values = append(values, arg)
	}
	return values
}
