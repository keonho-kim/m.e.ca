// Purpose: Track each declared external check against the latest possible file-change revision.
// Pattern: Pure state transitions.
// Usage: Runtime records CHECK outcomes; context and compaction preserve pending requirements.
// Related: internal/agent/run/action.go, internal/agent/compact/types.go, internal/agent/tool/execution.go.
package verification

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"slices"
	"unicode/utf8"
)

type Record struct {
	Key, Tool, Target, Status, Detail, CallID string
	Revision                                  int
}

func bounded(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	for !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit] + " [see original tool record]"
}

func Observe(previous []Record, call tool.Call, outcome tool.Result, revision int) ([]Record, error) {
	target := outcome.Target
	if call.Name == "execute_command" {
		var args struct{ Command string }
		if err := json.Unmarshal(call.Arguments, &args); err != nil {
			return nil, err
		}
		target = args.Command
	}
	if target == "" {
		target = string(call.Arguments)
	}
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(call.Name+"\n"+target)))
	record := Record{Key: key, Tool: call.Name, Target: bounded(target, 2048), Status: "not_run", CallID: call.ID, Revision: revision, Detail: bounded(outcome.Summary, 512)}
	if outcome.Process != nil && outcome.Process.Diagnostic != "" {
		record.Detail = bounded(outcome.Process.Diagnostic, 512)
	}
	if outcome.Started {
		record.Status = "passed"
		if outcome.Failed || outcome.Failure != nil {
			record.Status = "failed"
		}
		if call.Name == "execute_command" {
			if outcome.Process == nil || outcome.Process.Status != "exited" {
				record.Status = "unverified"
			} else if outcome.Process.ExitCode != 0 {
				record.Status = "failed"
			}
		}
	}
	records := slices.Clone(previous)
	for index := range records {
		if records[index].Key == key {
			records[index] = record
			return records, nil
		}
	}
	if len(records) >= 32 {
		return nil, fmt.Errorf("verification checklist exceeds 32 distinct checks; split the task without dropping pending requirements")
	}
	return append(records, record), nil
}

func Pending(records []Record, revision int) []Record {
	var pending []Record
	for _, record := range records {
		if record.Status != "passed" || record.Revision != revision {
			pending = append(pending, record)
		}
	}
	return pending
}

func Check(records []Record) error {
	if len(records) > 32 {
		return fmt.Errorf("verification checklist exceeds limit")
	}
	for _, record := range records {
		if len(record.Key) != 64 || record.Revision < 0 || len(record.Target) > 2100 || len(record.Detail) > 600 {
			return fmt.Errorf("invalid verification record")
		}
		switch record.Status {
		case "passed", "failed", "not_run", "unverified":
		default:
			return fmt.Errorf("invalid verification status")
		}
	}
	return nil
}
