// Purpose: Retain compact observed effects while original tool results remain in session history.
// Pattern: Ordered result projection.
// Usage: acceptResult updates these facts without completion judgements or filesystem rescans.
// Related: internal/agent/tool/facts_observation.go, internal/agent/record/checkpoint.go.
package run

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) observeResult(v validatedCall, outcome toolapi.Result) error {
	var command string
	if v.call.Name == "execute_command" {
		_ = json.Unmarshal(v.arguments["command"], &command)
	}
	op := toolapi.Operation{RunID: s.checkpoint.ID, CallID: v.call.ID, Name: v.call.Name, Revision: s.checkpoint.ChangeRevision, Started: outcome.Started, Versions: outcome.Versions, Changes: outcome.Changes, Process: outcome.Process, Command: command}
	s.checkpoint.Work.Observe(op)
	if len(outcome.Changes) > 0 {
		s.checkpoint.LastChange = excerpt(s.runner.Redact(outcome.Summary), 768)
	}
	return s.checkpoint.Work.Check()
}
