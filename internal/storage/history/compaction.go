// Purpose: Project conversation and actual outcomes into concise compaction inputs without duplicate audit events.
// Pattern: Pure history projection.
// Usage: SQLite passes ordered pages; originals remain untouched in session storage.
// Related: internal/agent/compact/types.go, internal/storage/sqlite/memory_sources.go.
package history

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/keonho-kim/v-code/internal/agent/compact"
)

func Compaction(page []Record) ([]compact.Record, error) {
	var records []compact.Record
	for _, record := range page {
		role, text := "", ""
		if record.Kind == "message" {
			m := record.Message
			switch m.Role {
			case "user", "assistant":
				role, text = m.Role, m.Content
			case "tool":
				role, text = "observation", toolExcerpt(m.Content)
			}
		} else {
			event := record.Event
			if op := event.Operation; op != nil {
				role = "outcome"
				text = op.Name + " " + op.Status
				if op.Process != nil {
					text += fmt.Sprintf("; process=%s exit=%d", op.Process.Status, op.Process.ExitCode)
				}
				if op.Command != "" {
					text += "\nCommand: " + op.Command
				}
				for _, change := range op.Changes {
					text += "\n" + change.Action + " " + change.Path
					if change.Destination != "" {
						text += " -> " + change.Destination
					}
				}
				if op.Step != nil {
					text += fmt.Sprintf("\nPlan step %d: %s", op.Step.Index+1, op.Step.Description)
				}
				if op.Failure != nil {
					text += "\nFailure: " + op.Failure.Code + ": " + op.Failure.Detail
					if op.Failure.Next != "" {
						text += "\nNext: " + op.Failure.Next
					}
				}
				if op.Process != nil && op.Process.Diagnostic != "" {
					text += "\nDiagnostic: " + op.Process.Diagnostic
				}
				if op.Result != "" {
					text += "\nResult: " + op.Result
				}
			} else if event.Kind == "tool_failed" || event.Kind == "denied" || event.Kind == "tool_interrupted" {
				role, text = "failure", event.Text
			} else if cp := event.Checkpoint; cp != nil && (cp.Status == "stopped" || cp.Status == "interrupted") {
				role, text = "failure", string(cp.Status)+": "+cp.Blocker+" "+cp.Observation
			}
		}
		if strings.TrimSpace(text) != "" {
			records = append(records, compact.Record{Sequence: record.Sequence, Role: role, Text: text})
		}
	}
	return records, nil
}

// Keep useful read/command observations, not JSON envelopes, anchors metadata or write arguments.
func toolExcerpt(text string) string {
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(text), &fields) != nil {
		return text
	}
	if nested := fields["Output"]; len(nested) > 0 {
		if json.Unmarshal(nested, &fields) != nil {
			return ""
		}
	}
	var parts []string
	for _, key := range []string{"Content", "Stdout", "Stderr"} {
		var value string
		if json.Unmarshal(fields[key], &value) == nil && strings.TrimSpace(value) != "" {
			parts = append(parts, key+": "+value)
		}
	}
	return strings.Join(parts, "\n")
}
