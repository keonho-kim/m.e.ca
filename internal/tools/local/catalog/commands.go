// Purpose: Declare command execution and owned-run inspection contracts.
// Pattern: Declarative tool contracts.
// Usage: The immutable catalog appends these command lifecycle definitions.
// Related: internal/tools/local/catalog/catalog.go, internal/tools/local/commands.go.
package catalog

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func commandSpecifications() []Specification {
	id := toolapi.Parameter{Name: "command_id", Description: "Opaque command_id returned by execute_command or list_command_runs in this session. Never an OS PID.", Kind: toolapi.Text, Required: true, MaxBytes: 128}
	wait := toolapi.Parameter{Name: "wait_seconds", Description: "Maximum time to wait for completion, in seconds. Default 180; 0 checks immediately. Expiration returns running and does not stop the command or renew its execution budget.", Kind: toolapi.Integer, Max: 86400}
	return []Specification{
		{"list_command_runs", "Listing commands", "List commands started by this session: active commands and completed results awaiting collection. Returns command_id, command, status, elapsed time and last output time. Does not inspect system processes or consume results.", nil, policy.ProcessControl},
		{"wait_command", "Waiting for command", "Wait for an owned command to finish, or return its running state when the wait expires. Returns current stdout/stderr excerpts, log paths and an exit code only after exit. Use the same command_id instead of executing the command again. Default wait is 180 seconds.", []toolapi.Parameter{id, wait}, policy.ProcessControl},
		{"cancel_command", "Cancelling command", "Cancel an owned command and its ordinary child process group, wait for cleanup, and return actual status and partial output. A command already finished returns its recorded result. Cancellation does not undo changes.", []toolapi.Parameter{id}, policy.ProcessControl},
	}
}
