// Purpose: Recognize explicit filesystem and destructive Git effects of common shell commands.
// Pattern: Command effect classification.
// Usage: Resolved calls feed operands into the shared mutation policy.
// Related: internal/policy/command/find.go, internal/policy/command/redirects.go, internal/policy/safety.go.
package command

import (
	"path/filepath"
	"strings"

	"github.com/keonho-kim/v-code/internal/policy"
)

func (a *analyzer) target(value word, s *state, removal bool) {
	if !value.known {
		a.result.Ask("Mutation target cannot be resolved statically")
		if value.glob {
			// Check a known directory prefix without expanding or scanning it.
			prefix := value.text
			if i := strings.IndexAny(prefix, "*?["); i >= 0 {
				prefix = prefix[:i]
			}
			prefix = filepath.Dir(prefix + "__glob_target__")
			a.target(word{text: filepath.Join(prefix, "__glob_target__"), known: true}, s, removal)
		}
		return
	}
	if value.text == "" {
		return
	}
	if !filepath.IsAbs(value.text) && s.dir == "" {
		a.result.Ask("Mutation working directory is unresolved")
		return
	}
	path, err := a.input.Resolve(s.dir, value.text, removal)
	if err != nil {
		a.result.Ask("Mutation target could not be canonicalized")
		return
	}
	decision, err := policy.Mutation(a.input.Project, a.input.Home, path, removal)
	if err != nil {
		a.err = err
		return
	}
	a.result.Merge(decision)
}

func (a *analyzer) effects(name string, args []word, s *state) {
	switch name {
	case "rm", "rmdir", "unlink":
		a.result.Ask("Deletion requires approval")
		for _, arg := range a.operands(args, nil) {
			a.target(arg, s, true)
		}
	case "mv", "cp", "install", "ln":
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
		if name == "mv" {
			a.result.Ask("Move requires approval")
			for _, arg := range values {
				a.target(arg, s, true)
			}
		}
	case "mkdir", "touch", "truncate", "tee":
		for _, arg := range a.operands(args, []string{"-m", "--mode", "-t", "-d", "--date", "-r", "--reference", "-s", "--size"}) {
			a.target(arg, s, false)
		}
	case "chmod", "chown", "chgrp":
		a.result.Ask("Permission or ownership changes require approval")
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
	case "sed":
		a.sed(args, s)
	case "dd", "mkfs", "mke2fs", "fdisk", "sfdisk", "parted", "wipefs", "mount", "umount", "chroot":
		a.result.Ask("System-level command requires explicit inspection")
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
