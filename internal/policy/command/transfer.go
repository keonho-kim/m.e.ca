// Purpose: Require approval for explicit outbound effects found by the shell AST.
// Pattern: Pure policy composition.
// Usage: Inspected calls and Git push pass resolved arguments here before approval is prepared.
// Related: internal/policy/command/transfer/inspect.go, internal/policy/command/transfer/destination.go, internal/policy/command/report.go.
package command

import (
	"slices"

	"github.com/keonho-kim/meca/internal/policy/command/transfer"
)

func (a *analyzer) transfer(name string, args []word) {
	values := make([]transfer.Argument, len(args))
	for i, arg := range args {
		values[i] = transfer.Argument{Value: arg.text, Known: arg.known && !arg.glob}
	}
	a.outbound(transfer.Inspect(name, values))
}

func (a *analyzer) outbound(inspection transfer.Inspection) {
	needsReview := inspection.Indirect
	for _, destination := range inspection.Destinations {
		if !inspection.Indirect && transfer.LoopbackHTTP(destination) {
			continue
		}
		needsReview = true
		target := transfer.Display(destination)
		if target != "" && a.current != nil && !slices.Contains(a.current.Targets, target) {
			a.current.Targets = append(a.current.Targets, target)
		}
	}
	if needsReview {
		a.result.UserRequired = true
		a.ask("Outbound data transfer requires approval")
	}
}
