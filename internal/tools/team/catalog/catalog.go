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
	reply := toolapi.Parameter{Name: "reply_to", Description: "Optional public sequence to reply to or follow up, including your earlier note. Links context, not permission or assignment.", Kind: toolapi.Integer, Max: 1<<53 - 1}
	return []toolapi.Definition{
		{Name: "add_team_member", Description: "Add one teammate for required, unfinished work only when no owner or recent invitation covers it, its scope and prerequisites let it start independently now, and current members including you have other work so another peer enables substantial parallel progress. Invite before taking every independent part yourself. Use an available active peer first; avoid duplicate checks, overlapping edits or waiting on the same unfinished prerequisite. Omit member_name for a new named session or supply an inactive name to restore relevant history. An active name returns status without assigning or waking that peer. Up to eight members may be active. A full team returns availability; continue useful work without retrying unchanged capacity.", Parameters: []toolapi.Parameter{text("message", "Required result, exact paths or inspection area, boundaries, ready dependencies and interfaces, and why another peer helps now.", true), text("member_name", "Optional inactive member's exact name from list_team_members.", false)}},
		{Name: "leave_team", Description: "Finish your contribution and close your session after this native result is saved. First settle your running commands and review relevant feedback received since your last model decision. Tell peers what changed, actual checks and remaining issues. If you are the final peer, the result instead asks you to report directly to the user; answer normally. This must be the final call in a batch.", Parameters: []toolapi.Parameter{text("message", "Your contribution, observed checks and remaining concerns.", true)}},
		{Name: "list_team_members", Description: "List teammate names, active status, all currently owned paths, brief contribution summaries and shared behavior. Read the complete declaration with read_team_messages using after equal to the work sequence minus one. Use a closed member's name with add_team_member to restore their session. Private history and reasoning remain private.", Parameters: []toolapi.Parameter{text("after_member", "Last member name from a preceding page; omit for the first page.", false)}},
		{Name: "read_team_messages", Description: "Read a chronological page of committed public conversation after a sequence. New messages are supplied automatically at decision boundaries; use this for older context. Continue from the final returned sequence.", Parameters: []toolapi.Parameter{{Name: "after", Description: "Last known public sequence; omit or use zero for the beginning.", Kind: toolapi.Integer, Max: 1<<53 - 1}}},
		{Name: "set_work", Description: "Record or replace your agreed contribution before changing its files/subdirectories. State the outcome and shared interfaces/behavior in contract. Success records your scope, not peer acceptance; resolve missing agreement with discuss. Overlap identifies the owner/path and retains your prior scope: coordinate a handoff or choose independent work. Empty paths release ownership, not mark completion. Settle active commands before changing paths; same-scope summary/contract updates are allowed. Submit before dependent edits.", Parameters: []toolapi.Parameter{text("summary", "Your agreed responsibility or current result in the user's language, related to peers' actual work.", true), {Name: "paths", Description: "Exact file/subdirectory paths relative to project_root, or absolute paths. No globs or whole project root. Empty list releases your current scope.", Kind: toolapi.TextList, Required: true, AllowEmpty: true, Max: 64, MaxBytes: 4096}, text("contract", "Optional shared interfaces/behavior and relevant agreed Spec reference. A declaration alone is not peer acceptance.", false), reply}},
		{Name: "discuss", Description: "Publish a role or Spec proposal/acceptance, question, answer, objection, alternative or finding to all teammates. Use reply_to for the addressed note and mentions for affected peers; everyone can read and challenge it. Returns after publication without waiting, assigning work, interrupting generation or starting sessions. Agreement requires the relevant peers' acceptance. Include concrete responsibilities, evidence and paths as needed. Keep routine progress and empty acknowledgments in ordinary responses; avoid duplicate text. Continue ready agreed work or read-only inspection while your role is unresolved.", Parameters: []toolapi.Parameter{text("message", "Your concise role/Spec proposal, acceptance, feedback, question or evidence in the user's language.", true), reply, {Name: "mentions", Description: "Optional exact names of other teammates whose attention is useful. Never include your own name. Omit unless particular other teammates need attention. The message remains public; inactive teammates are not restarted.", Kind: toolapi.TextList, AllowEmpty: true, Max: 8, MaxBytes: 4096}}},
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
