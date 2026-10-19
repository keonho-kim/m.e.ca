// Purpose: Inspect redirect destinations and executable expansions without treating literal file text as commands.
// Pattern: Shell syntax analysis.
// Usage: Each statement checks redirects before inspecting its command.
// Related: internal/policy/command/flow.go, internal/policy/command/words.go.
package command

import "mvdan.cc/sh/v3/syntax"

func (a *analyzer) redirect(r *syntax.Redirect, s *state) {
	value := a.word(r.Word, s, true)
	if r.Hdoc != nil {
		a.substitutions(r.Hdoc, s)
	}
	switch r.Op {
	case syntax.RdrOut, syntax.AppOut, syntax.RdrInOut, syntax.RdrClob, syntax.RdrAll, syntax.AppAll:
		if len(value) > 0 {
			a.target(value[0], s, false)
		}
	case syntax.DplOut:
		// POSIX >& duplicates a descriptor; Bash also accepts file targets.
		if len(value) > 0 && !descriptor(value[0]) {
			a.target(value[0], s, false)
		}
	}
}

func descriptor(w word) bool {
	if !w.known {
		return false
	}
	if w.text == "-" {
		return true
	}
	for _, r := range w.text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return w.text != ""
}
