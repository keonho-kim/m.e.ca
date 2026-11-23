// Purpose: Declare team-only native communication and history lookup tools.
// Pattern: Immutable tool catalog.
// Usage: The cooperative executor adds these definitions to the ordinary local tools.
// Related: internal/tools/team/executor.go, internal/tools/team/coordination.go.
package catalog

import (
	"slices"

	"github.com/keonho-kim/meca/internal/agent/team"
	"github.com/keonho-kim/meca/internal/agent/team/source"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

var catalog = buildDefinitions()

func buildDefinitions() []toolapi.Definition {
	text := func(name, description string, required bool) toolapi.Parameter {
		return toolapi.Parameter{Name: name, Description: description, Kind: toolapi.Text, Required: required, MaxBytes: 64 * 1024 * 1024}
	}
	reply := toolapi.Parameter{Name: "reply_to", Description: "Optional public sequence to reply to or follow up, including your earlier note. Links context, not permission or assignment.", Kind: toolapi.Integer, Max: 1<<53 - 1}
	domain := toolapi.Parameter{Name: "domain", Description: "Stable shared key for the independent work domain, such as api-compatibility. Reuse the same key for the same work.", Kind: toolapi.SingleLineText, Required: true, Max: source.MaxWorkDomainCharacters}
	workRef := toolapi.Parameter{Name: "work_ref", Description: "Optional sequence of the public discuss proposal for this exact work. Reuse it across invitations and declarations even if the labels differ.", Kind: toolapi.Integer, Max: source.MaxWorkRef}
	return []toolapi.Definition{
		{
			Name: "add_team_member",
			Description: "Invite one teammate for required work that can start independently now and has no owner or pending invitation. " +
				"Review active teammates' current work, command progress and blockers in the team overview and public conversation. Include your own workload. " +
				"Offer the work to an active teammate who can start without delaying their other work. If a likely candidate's availability is unclear, ask them with discuss and mentions and wait for their reply before inviting. " +
				"Invite when no active teammate can take it now and your work can continue in parallel. " +
				"Omit member_name for a new named session or name an inactive teammate to restore their history. An active name returns status without assigning work or waking that peer. " +
				"Reuse an existing domain key or public work_ref. If the work is already owned, an invitation is pending, or resources are unavailable, continue useful work within the active team.",
			Parameters: []toolapi.Parameter{
				domain, workRef,
				text("message", "Expected result, exact paths or inspection area, boundaries, ready prerequisites, relevant interfaces and why no active teammate can take it now.", true),
				text("member_name", "Optional inactive member's exact name from list_team_members.", false),
			},
		},
		{Name: "leave_team", Description: "Finish your contribution and close your session after this native result is saved. First settle your running commands and review relevant feedback received since your last model decision. Tell peers what changed, actual checks and remaining issues. If you are the final peer, the result instead asks you to report directly to the user; answer normally. This must be the final call in a batch.", Parameters: []toolapi.Parameter{text("message", "Your contribution, observed checks and remaining concerns.", true)}},
		{Name: "list_team_members", Description: "List teammate names, active status, invitation domains, declared domains and public work references, owned paths, contribution summaries and shared behavior. Read the complete declaration with read_team_messages using after equal to the work sequence minus one. Use a closed member's name with add_team_member to restore their session. Private history and reasoning remain private.", Parameters: []toolapi.Parameter{text("after_member", "Last member name from a preceding page; omit for the first page.", false)}},
		{Name: "read_team_messages", Description: "Read a chronological page of committed public conversation after a sequence. New messages are supplied automatically at decision boundaries; use this for older context. Continue from the final returned sequence.", Parameters: []toolapi.Parameter{{Name: "after", Description: "Last known public sequence; omit or use zero for the beginning.", Kind: toolapi.Integer, Max: 1<<53 - 1}}},
		{
			Name: "set_work",
			Description: "Record your agreed contribution before changing files. Use the same domain and work_ref as the shared proposal or invitation. Coordinate acceptance with affected peers through discuss. " +
				"A conflict identifies the owner and preserves your prior scope. Coordinate a handoff or choose independent work. Settle active commands before changing paths. Submit before dependent edits.",
			Parameters: []toolapi.Parameter{
				text("summary", "Your agreed responsibility or current result in the user's language, related to peers' actual work.", true),
				{Name: "paths", Description: "Exact file or directory paths. Use project-relative or absolute paths. Empty releases paths. No globs or whole project root.", Kind: toolapi.TextList, Required: true, AllowEmpty: true, Max: 64, MaxBytes: 4096},
				{Name: "domains", Description: "All work domains you currently own, including read-only work. Omit to retain current domains. An empty list releases them. Report completion separately.", Kind: toolapi.ObjectList, AllowEmpty: true, Max: source.MaxWorkDomains, Fields: []toolapi.Parameter{domain, workRef}},
				text("contract", "Optional shared interfaces, behavior and relevant agreed Spec reference.", false),
				reply,
			},
		},
		{Name: "discuss", Description: "Publish a role or Spec proposal/acceptance, question, answer, objection, alternative or finding to all teammates. Use reply_to for the addressed note and mentions for affected peers; everyone can read and challenge it. Returns after publication without waiting, assigning work, interrupting generation or starting sessions. Agreement requires the relevant peers' acceptance. Include concrete responsibilities, evidence and paths as needed. Keep routine progress and empty acknowledgments in ordinary responses; avoid duplicate text. Continue ready agreed work or read-only inspection while your role is unresolved.", Parameters: []toolapi.Parameter{text("message", "Your concise role/Spec proposal, acceptance, feedback, question or evidence in the user's language.", true), reply, {Name: "mentions", Description: "Optional exact names of other teammates whose attention is useful. Never include your own name. Omit unless particular other teammates need attention. The message remains public; inactive teammates are not restarted.", Kind: toolapi.TextList, AllowEmpty: true, Max: team.MaxMentions, MaxBytes: 4096}}},
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
		for j := range out[i].Parameters {
			out[i].Parameters[j].Fields = slices.Clone(out[i].Parameters[j].Fields)
		}
	}
	return out
}
