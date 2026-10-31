// Purpose: Expose owned command lifecycle, completion facts and management guidance to the runtime.
// Pattern: Consumer-owned resource lifecycle contract.
// Usage: The runtime binds commands to one request and commits completed observations serially.
// Related: internal/tools/local/commands.go, internal/agent/run/commands.go.
package tool

import (
	"context"
	"encoding/json"
	"time"
)

// CommandManagementHint guides subsequent native calls for an existing running command.
const CommandManagementHint = "Use wait_command with this command_id to wait or inspect output (wait_seconds=0 checks immediately), cancel_command with this command_id to stop it, or list_command_runs with no arguments to list session commands and recover their IDs. Waiting does not stop the command. Do not execute the same command again just to check progress."

type CommandRun struct {
	CallID           string "json:\"-\""
	ID               string "json:\"command_id\""
	Command          string "json:\"command\""
	Status           string "json:\"status\""
	ExitCode         *int   "json:\"exit_code,omitempty\""
	ElapsedMillis    int64  "json:\"elapsed_millis\""
	LastOutputMillis *int64 "json:\"last_output_millis\""
}

type CommandCompletion struct {
	ID, CallID, Command string
	Result              Result
}

type CommandExecutor interface {
	BeginCommands(context.Context, time.Time)
	StopCommands()
	CommandActivity() []CommandRun
	CollectCommands() ([]CommandCompletion, error)
	CommandCapture(string) (json.RawMessage, error)
	AcknowledgeCommand(string) error
}

// CommandNotifier signals terminal changes without exposing operating-system PIDs.
// The channel coalesces completions; the runtime still collects every result.
type CommandNotifier interface{ CommandUpdates() <-chan struct{} }
