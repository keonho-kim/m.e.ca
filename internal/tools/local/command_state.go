// Purpose: Preserve command identity when projection or archival fails after admission.
// Pattern: Typed process-state to tool-result adapter.
// Usage: Execute and command controls use the same actual-state error projection.
// Related: internal/tools/local/command.go, internal/tools/local/command_control.go.
package local

import (
	"errors"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/local/process"
)

func (e *Executor) commandResult(state process.State) (toolapi.Result, error) {
	if toolapi.CommandActive(state.Status) {
		return e.runningCommand(state)
	}
	result, err := e.readCommandResult(state.ID)
	if state.Error != "" {
		err = errors.Join(err, faults.Errorf(diagcatalog.LocalCommandResultArchiveFailedFor, state.ID, state.Error))
	}
	err = errors.Join(err, state.Failure)
	if result.Process == nil {
		result.Started = state.Failure == nil
		result.Process = &toolapi.ProcessOutcome{CommandID: state.ID, Status: state.Status, ExitCode: state.ExitCode}
	}
	result.Process.Command = e.commandLabel(state.ID, state.Command)
	if err != nil {
		result.Failed = true
		if result.Failure == nil {
			result.Failure = faults.FailureOf(err)
		}
	}
	return result, err
}
