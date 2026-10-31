// Purpose: Project conversation and actual outcomes into concise compaction inputs without duplicate audit events.
// Pattern: Pure history projection.
// Usage: SQLite passes ordered pages; originals remain untouched in session storage.
// Related: internal/agent/compact/types.go, internal/storage/sqlite/memory_sources.go.
package history

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/compact"
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
				if m.Incomplete {
					role = "unfinished_assistant_draft"
				}
			case "tool":
				role, text = "observation", toolExcerpt(m.Content)
			}
		} else {
			event := record.Event
			if op := event.Operation; op != nil {
				role = "outcome"
				text = op.Name + " " + op.Status
				if op.Process != nil {
					text += "; process=" + op.Process.Status
					if op.Process.CommandID != "" {
						text += "; command_id=" + op.Process.CommandID
					}
					if op.Process.Status != "running" {
						text += fmt.Sprintf(" exit=%d", op.Process.ExitCode)
					}
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
				if event.Kind == "command_finished" {
					if output := toolExcerpt(string(event.Output)); output != "" {
						text += "\nOutput: " + output
					}
				}
			} else if event.Kind == "lat_feedback" {
				role, text = "review_observation", event.Text
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

// Keep useful read/command observations, not JSON envelopes or write arguments.
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
