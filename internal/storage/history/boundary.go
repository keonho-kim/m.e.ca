// Purpose: Retain a recent tail without separating native calls from their results.
// Pattern: Incremental reverse-history selection.
// Usage: Storage adapters feed newest messages first until Keep returns a safe prefix boundary.
// Related: internal/storage/sqlite/memory_boundary.go, internal/agent/model/message.go.
package history

import (
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

// TailSelection treats the byte target as a selection budget, not a message limit.
type TailSelection struct {
	Target       int
	used         int
	pending      map[string]bool
	failed       map[string]bool
	repair       map[string]bool
	lastBoundary int64
	hasBoundary  bool
}

func (s *TailSelection) Keep(sequence int64, size int, message modelapi.Message) (int64, bool) {
	s.used += size
	if s.pending == nil {
		s.pending = map[string]bool{}
		s.failed = map[string]bool{}
		s.repair = map[string]bool{}
	}
	if message.Role == "tool" && message.ToolCallID != "" {
		s.pending[message.ToolCallID] = true
		if message.ToolName == "edit_file" {
			var body struct{ Failure json.RawMessage }
			if json.Unmarshal([]byte(message.Content), &body) == nil && len(body.Failure) > 0 && string(body.Failure) != "null" {
				s.failed[message.ToolCallID] = true
				// Preserve the correction's source even if a later draft fit alone.
				s.hasBoundary = false
			}
		}
	}
	if message.Role == "assistant" {
		for _, call := range message.Calls {
			delete(s.pending, call.ID)
			if call.Name == "execute_command" {
				clear(s.repair)
			}
			var args struct{ Path string }
			if json.Unmarshal(call.Arguments, &args) == nil && args.Path != "" {
				if s.failed[call.ID] {
					s.repair[args.Path] = true
					delete(s.failed, call.ID)
				}

			}
		}
	}
	// A tool result is never the first retained record, even for an orphan.
	// Missing earlier reads do not retain an unbounded repair prefix. Native pairs
	// remain indivisible even when this selection budget is exceeded.
	if s.used >= max(s.Target, 16*1024*1024) {
		clear(s.repair)
	}
	if len(s.pending) == 0 && len(s.repair) == 0 && message.Role != "tool" {
		if s.used >= s.Target {
			if s.hasBoundary {
				return s.lastBoundary, true
			}
			return sequence - 1, true
		}
		// Prefer a complete recent tail over an older group that exceeds the
		// selection target. The newest group itself may still exceed that target.
		s.lastBoundary, s.hasBoundary = sequence-1, true
	}
	return 0, false
}
