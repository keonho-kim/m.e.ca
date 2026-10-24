// Purpose: Associate mandatory shell reviews with the inspected command and its actual targets.
// Pattern: Analysis report values.
// Usage: Local command preparation converts findings into redacted approval context.
// Related: internal/policy/command/analyze.go, internal/tools/local/command_safety.go, internal/policy/safety.go.
package command

import (
	"slices"

	"github.com/keonho-kim/meca/internal/policy"
)

type Finding struct {
	Command          string
	Reasons, Targets []string
}

type Report struct {
	Safety   policy.Safety
	Findings []Finding
}

func (a *analyzer) finding(name string) func() {
	previous := a.current
	current := &Finding{Command: name}
	a.current = current
	return func() {
		if len(current.Reasons) > 0 {
			a.findings = append(a.findings, *current)
		}
		a.current = previous
	}
}

func (a *analyzer) ask(reason string) {
	a.result.Ask(reason)
	if a.current == nil {
		a.findings = append(a.findings, Finding{Command: "/bin/sh", Reasons: []string{reason}})
		return
	}
	if !slices.Contains(a.current.Reasons, reason) {
		a.current.Reasons = append(a.current.Reasons, reason)
	}
}

func (a *analyzer) merge(decision policy.Safety) {
	for _, reason := range decision.Reasons {
		a.ask(reason)
	}
	a.result.Merge(decision)
	if a.current != nil {
		for _, path := range decision.Targets {
			if !slices.Contains(a.current.Targets, path) {
				a.current.Targets = append(a.current.Targets, path)
			}
		}
	}
}
