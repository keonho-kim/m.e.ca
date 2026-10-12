// Purpose: Preserve runtime-reported task states and exact changed-file metadata beside model summaries.
// Pattern: Pure projection.
// Usage: Merged for each source page and supplied as non-authoritative context facts.
// Related: internal/agent/compact/types.go, internal/agent/compact/controller.go.
package compact

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

func (l *Ledger) merge(record Record) {
	if record.Task != nil {
		fact := *record.Task
		fact.Source = record.Sequence
		for i := range l.Tasks {
			if l.Tasks[i].ID == fact.ID {
				l.Tasks[i] = fact
				return
			}
		}
		l.Tasks = append(l.Tasks, fact)
	}
	if record.Operation != nil {
		op := *record.Operation
		op.Source = record.Sequence
		l.Operations = append(l.Operations, op)
	}
}
func (m Memory) Context() (string, error) {
	if m.Revision == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString(m.Markdown)
	b.WriteString("\n\n## Runtime evidence ledger\n")
	quote := func(text string) string { value, _ := json.Marshal(text); return string(value) }
	seenPlans := map[string]bool{}
	for _, task := range m.Ledger.Tasks {
		fmt.Fprintf(&b, "- Task %s: status=%s; phase=%s; goal source=#%d\n", quote(task.ID), quote(task.Status), quote(task.Phase), task.Source)
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
		if op.Command != "" {
			fmt.Fprintf(&b, "  - Command: %s\n", quote(op.Command))
		}
		for _, change := range op.Changes {
			fmt.Fprintf(&b, "  - File: %s; action=%s; destination=%s\n", quote(change.Path), quote(change.Action), quote(change.Destination))
		}
	}
	return b.String(), nil
}
func (m Memory) Check(maxSummary int) error {
	if m.Revision < 0 || m.Through < 0 || m.UsageThrough < 0 || len(m.Markdown) > maxSummary || !utf8.ValidString(m.Markdown) {
		return fmt.Errorf("invalid compaction memory")
	}
	ledger, err := json.Marshal(m.Ledger)
	if err != nil || len(ledger) > 4*1024*1024 {
		return fmt.Errorf("compaction ledger exceeds limit; originals were retained")
	}
	return nil
}

var sections = []string{"Goals", "Task history", "Steps and changed files", "Verification", "Decisions and constraints", "Pending work and blockers", "Continuation"}

func checkSummary(text string, maxBytes int) error {
	if len(text) > maxBytes || !utf8.ValidString(text) {
		return fmt.Errorf("compaction summary exceeds its limit")
	}
	index := -1
	bullets := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			if index >= 0 && bullets == 0 {
				return fmt.Errorf("summary section has no items")
			}
			index++
			bullets = 0
			if index >= len(sections) || line != "## "+sections[index] {
				return fmt.Errorf("unexpected compaction section")
			}
			continue
		}
		if index < 0 || !strings.HasPrefix(line, "- ") {
			return fmt.Errorf("compaction summary must contain headings and bullet items only")
		}
		bullets++
	}
	if index != len(sections)-1 || bullets == 0 {
		return fmt.Errorf("compaction summary is missing required sections")
	}
	return nil
}
