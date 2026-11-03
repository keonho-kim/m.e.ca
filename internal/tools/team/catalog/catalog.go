// Purpose: Declare team-only native communication and history lookup tools.
// Pattern: Immutable tool catalog.
// Usage: The cooperative executor adds these definitions to the ordinary local tools.
// Related: internal/tools/team/executor.go, internal/tools/team/coordination.go.
package catalog

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"slices"
)

var catalog = buildDefinitions()

func buildDefinitions() []toolapi.Definition {
	text := func(name, description string, required bool) toolapi.Parameter {
		return toolapi.Parameter{Name: name, Description: description, Kind: toolapi.Text, Required: required, MaxBytes: 64 * 1024 * 1024}
	}
	return []toolapi.Definition{
		{Name: "call_friend", Description: "Invite a peer for a useful independent contribution. Omit agent_id to create a new private session; provide a known ID to restore its history. An already active peer keeps working without another instruction. At most eight sessions can be active; a full room returns availability without creating another session. Do not repeatedly retry a full room.", Parameters: []toolapi.Parameter{text("message", "State the distinct task, exact project-relative target files, boundaries and necessary context.", true), text("agent_id", "Optional closed peer ID to restore.", false)}},
		{Name: "say_goodbye", Description: "Finish your contribution and close your session after this native result is saved. First settle your running commands. Tell peers what changed, actual checks and remaining issues. If you are the final peer, the result instead asks you to report directly to the user; answer normally, or seek further help if justified. This must be the final call in a batch.", Parameters: []toolapi.Parameter{text("message", "Your contribution, observed checks and remaining concerns.", true)}},
		{Name: "list_friends", Description: "List stable peer IDs, their last contribution and whether they are active. Use an ID with call_friend to restore a closed peer. Returns metadata without private conversation or reasoning.", Parameters: []toolapi.Parameter{text("after_peer", "Last peer ID from a preceding page; omit for the first page.", false)}},
		{Name: "read_team_messages", Description: "Read a chronological page of committed public conversation after a sequence. New messages are supplied automatically at decision boundaries; use this for older context. Continue from the final returned sequence.", Parameters: []toolapi.Parameter{{Name: "after", Description: "Last known public sequence; omit or use zero for the beginning.", Kind: toolapi.Integer, Max: 1 << 30}}},
	}
}

// IsTool classifies the exact immutable native catalog without loading executors.
func IsTool(name string) bool { _, ok := definition(name); return ok }

func definition(name string) (toolapi.Definition, bool) {
	for _, item := range catalog {
		if item.Name == name {
			return item, true
		}
	}
	return toolapi.Definition{}, false
}
func Definitions() []toolapi.Definition {
	out := slices.Clone(catalog)
	for i := range out {
		out[i].Parameters = slices.Clone(out[i].Parameters)
	}
	return out
}
