// Purpose: Preserve command provenance as a failed observation when its result archive is unavailable.
// Pattern: Terminal failure projection without replaying process effects.
// Usage: Completion collection and command controls share this bounded fallback.
// Related: internal/tools/local/commands.go, internal/tools/local/command_state.go.
package local

import (
	"encoding/json"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (e *Executor) unavailableCommandResult(id, command, status string, exit int, started bool, detail string) toolapi.Result {
	failure := &fault.Failure{Code: diag.PersistenceFailed,
		Detail: diag.Text(diag.LocalCommandResultArchiveFailedFor, id, e.redact(detail)),
		Next:   diag.Text(diag.CommandCaptureUnavailable)}
	// A failed output archive does not change the observed process exit.
	body, _ := json.Marshal(struct {
		CommandID         string `json:"command_id"`
		Status            string
		ExitCode          int
		Truncated         bool
		OutputUnavailable bool `json:"output_unavailable"`
	}{id, status, exit, true, true})
	return toolapi.Result{Started: started, Failed: true, Failure: failure,
		Summary: diag.Text(diag.CommandCaptureUnavailable), Content: body,
		Process: &toolapi.ProcessOutcome{CommandID: id, Command: e.commandLabel(id, command),
			Status: status, ExitCode: exit, Truncated: true, Diagnostic: failure.Detail}}
}
