// Purpose: Include ancestor removals requested by rmdir parent options in the common path policy.
// Pattern: Command-specific operand analysis.
// Usage: The removal effect classifier checks each explicit parent without scanning directories.
// Related: internal/policy/command/effects.go, internal/policy/command/targets.go.
package command

import "strings"

func removalParents(args []word) bool {
	for _, arg := range args {
		if arg.known && arg.text == "--" {
			break
		}
		if arg.known && (arg.text == "--parents" || strings.HasPrefix(arg.text, "-") && !strings.HasPrefix(arg.text, "--") && strings.Contains(arg.text[1:], "p")) {
			return true
		}
	}
	return false
}

func (a *analyzer) removalParents(value word, s *state) {
	if !value.known {
		return // target already requires approval for the unresolved operand.
	}
	path := strings.TrimRight(value.text, "/")
	for a.err == nil {
		i := strings.LastIndex(path, "/")
		if i < 0 {
			return
		}
		path = strings.TrimRight(path[:i], "/")
		if path == "" {
			path = "/"
		}
		a.target(word{text: path, known: true}, s, true)
		if path == "/" {
			return
		}
	}
}
