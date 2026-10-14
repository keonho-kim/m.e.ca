// Purpose: Validate stored memory and format its summary and runtime evidence for context.
// Pattern: Memory projection boundary.
// Usage: Runtime requests context and persistence validates memory revisions.
// Related: internal/agent/compact/types.go, internal/agent/compact/controller.go.
package compact

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/verification"
	"strings"
	"unicode/utf8"
)

func (m Memory) Context() (string, error) {
	if m.Revision == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString(m.Markdown)
	b.WriteString("\n\n## Runtime evidence ledger\n")
	quote := func(text string) string { value, _ := json.Marshal(text); return string(value) }
	for _, fact := range m.Ledger.Goals {
		body, err := json.Marshal(fact.State)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "- Goal state source=#%d: %s\n", fact.Source, body)
	}
	for _, fact := range m.Ledger.Reviews {
		body, err := json.Marshal(fact.Result)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "- Conclave ballot source=#%d (review only; not execution): %s\n", fact.Source, body)
	}
	seenPlans := map[string]bool{}
	for _, task := range m.Ledger.Tasks {
		fmt.Fprintf(&b, "- Task %s: status=%s; phase=%s; goal source=#%d\n", quote(task.ID), quote(task.Status), quote(task.Phase), task.Source)
		for _, check := range task.Checks {
			fmt.Fprintf(&b, "  - Check %s: status=%s; revision=%d/current=%d; call=%s; detail=%s\n", quote(check.Target), quote(check.Status), check.Revision, task.ChangeRevision, quote(check.CallID), quote(check.Detail))
		}
		if task.Step == nil {
			fmt.Fprintf(&b, "  - Original goal: %s\n", quote(task.Objective))
		} else {
			key := fmt.Sprintf("%s:%d", task.Step.PlanID, task.Step.Revision)
			if !seenPlans[key] {
				fmt.Fprintf(&b, "  - Original Plan goal: %s\n", quote(task.Step.Objective))
				seenPlans[key] = true
			}
		}
		if task.Step != nil {
			fmt.Fprintf(&b, "  - Plan %s revision %d step %d: %s\n", quote(task.Step.PlanID), task.Step.Revision, task.Step.Index+1, quote(task.Step.Description))
		}
	}
	for _, op := range m.Ledger.Operations {
		fmt.Fprintf(&b, "- Operation %s: task=%s; tool=%s; status=%s; check=%t; source=#%d\n", quote(op.CallID), quote(op.TaskID), quote(op.Name), quote(op.Status), op.Check, op.Source)
		if op.Process != nil && op.Process.Diagnostic != "" {
			fmt.Fprintf(&b, "  - Check diagnostic: %s; capture_truncated=%t\n", quote(op.Process.Diagnostic), op.Process.Truncated)
		}
		if op.Failure != nil {
			fmt.Fprintf(&b, "  - Failure: %s; next=%s\n", quote(op.Failure.Code+": "+op.Failure.Detail), quote(op.Failure.Next))
		}
		if op.Command != "" {
			fmt.Fprintf(&b, "  - Command: %s\n", quote(op.Command))
		}
		for _, change := range op.Changes {
			if lines := change.Lines; lines != nil {
				fmt.Fprintf(&b, "  - Lines: before=%d-%d; after=%d-%d; removed=%t (positions at execution time)\n", lines.OldStartLine, lines.OldEndLine, lines.NewStartLine, lines.NewEndLine, lines.Removed)
			}
			fmt.Fprintf(&b, "  - File: %s; action=%s; destination=%s\n", quote(change.Path), quote(change.Action), quote(change.Destination))
		}
	}
	b.WriteString("\n## Self-grounding records (model claims, not execution evidence)\n")
	for _, fact := range m.Ledger.Groundings {
		fmt.Fprintf(&b, "- Source #%d: %s\n", fact.Source, quote(fact.Record.Markdown()))
	}
	return b.String(), nil
}
func (m Memory) Check(maxSummary int) error {
	if m.Revision < 0 || m.Through < 0 || m.UsageThrough < 0 || len(m.Markdown) > maxSummary || !utf8.ValidString(m.Markdown) {
		return fmt.Errorf("invalid compaction memory")
	}
	for _, fact := range m.Ledger.Goals {
		if err := fact.State.Check(); err != nil {
			return err
		}
	}
	for _, fact := range m.Ledger.Reviews {
		if err := fact.Result.Check(); err != nil {
			return err
		}
	}
	for _, fact := range m.Ledger.Groundings {
		if err := fact.Record.Check(); err != nil {
			return err
		}
	}
	for _, task := range m.Ledger.Tasks {
		if err := verification.Check(task.Checks); err != nil {
			return err
		}
	}
	ledger, err := json.Marshal(m.Ledger)
	if err != nil || len(ledger) > 4*1024*1024 {
		return fmt.Errorf("compaction ledger exceeds limit; originals were retained")
	}
	return nil
}
