// Purpose: Admit announced peer work and reserve its actual shared-workspace targets.
// Pattern: Tool-to-resource adapter.
// Usage: Executor.Prepare checks public participation before constructing large local candidates.
// Related: internal/agent/team/peer.go, internal/tools/team/executor.go, internal/tools/team/workspace/workspace.go.
package team

import (
	"context"
	"fmt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/team/workspace"
)

func (e *Executor) reserve(ctx context.Context, call toolapi.Call) (*workspace.Lease, []string, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	switch call.Name {
	case "wait_command", "cancel_command", "list_command_runs":
		return nil, nil, nil
	}
	if !e.Peer.HasSpoken() {
		return nil, nil, &toolapi.Failure{
			Code:   "team_contribution_required",
			Detail: "Nothing executed. Tell your teammates your contribution before starting local work.",
			Next:   "In your public response, state what you will inspect or change and the project-relative paths. Refer to a peer's announced work when it affects yours. You may include the tool call in the same response. Keep the message concise and use the user's language.",
		}
	}
	if call.Name == "view_image" {
		return nil, nil, nil
	}
	if call.Name == "execute_command" {
		// Mutation targets are available after local shell analysis. Reserve them
		// at execution, so freezing a batch cannot wait on its own pending calls.
		return nil, nil, e.Peer.CheckUserDecision()
	}
	if id := e.Local.ActiveCommand(); id != "" {
		return nil, nil, &toolapi.Failure{Code: "team_command_pending", Detail: "Your command " + id + " is still running; this file change did not start.", Next: "Use wait_command or cancel_command with this command_id before changing files. Independent commands remain available within your command capacity; use list_command_runs to inspect them."}
	}
	paths, err := e.paths(call)
	if err != nil {
		return nil, nil, err
	}
	if err := e.Peer.CheckWork(paths); err != nil {
		return nil, nil, err
	}
	held, err := e.reservePaths(ctx, call, paths, false)
	return held, paths, err
}

func (e *Executor) reservePaths(ctx context.Context, call toolapi.Call, paths []string, command bool) (*workspace.Lease, error) {
	obsolete, err := e.Peer.UserChanges()
	if err != nil {
		return nil, err
	}
	waiting := func() error {
		return e.Peer.Emit(runrecord.Event{Kind: "tool_waiting", CallID: call.ID, Call: &call, Text: "Waiting for shared workspace access"})
	}
	if command {
		return e.Workspace.ReserveCommand(ctx, paths, obsolete, nil)
	}
	return e.Workspace.ReserveFiles(ctx, paths, obsolete, waiting)
}

func (e *Executor) paths(call toolapi.Call) ([]string, error) {
	input, err := toolapi.DisplayArgs(call.Arguments)
	if err != nil {
		return nil, err
	}
	paths := input.Paths
	if input.Path != "" {
		paths = append(paths, input.Path)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("file operation has no target")
	}
	return e.canonicalPaths(paths, false)
}
