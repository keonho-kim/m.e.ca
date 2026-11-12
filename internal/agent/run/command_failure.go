// Purpose: Preserve failed original-capture observations without replaying or permanently retaining a command.
// Pattern: Pure failure projection before the journal durability boundary.
// Usage: Immediate and background command completion paths record this result before acknowledgement.
// Related: internal/agent/run/commands.go, internal/agent/run/execution_result.go.
package run

import (
	"encoding/json"

	"github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *execution) failedCommandCapture(result tool.Result, id string, cause error) (tool.Result, error) {
	result.Failed = true
	result.Capture = nil
	result.Failure = &fault.Failure{Code: diag.PersistenceFailed, Detail: diag.Text(diag.CommandCaptureFailed, id, s.runner.Redact(cause.Error())), Next: diag.Text(diag.CommandCaptureUnavailable), Cause: cause}
	result.Summary = result.Failure.Detail
	if result.Process != nil {
		process := *result.Process
		process.Truncated, process.Diagnostic = true, result.Failure.Detail
		result.Process = &process
	}
	// Keep the existing excerpt and observed fields inside an explicit failure
	// envelope; a missing original must never look like successful empty output.
	body, err := json.Marshal(struct {
		Failure *fault.Failure
		Output  json.RawMessage
	}{result.Failure, result.Content})
	if err != nil {
		return result, err
	}
	result.Content = body
	return result, nil
}
