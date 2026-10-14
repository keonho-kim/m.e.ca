// Purpose: Compose project-root instructions with runtime invariants and a small sequential tool protocol.
// Pattern: Pure Function.
// Usage: Rebuilt from one instruction snapshot for every fresh Ralph iteration.
// Related: internal/agent/prompt/prompt_test.go, internal/agent/run/context.go.
package prompt

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxInstructionsBytes = 32 * 1024

func Compose(base, instructions string, planning bool) (string, error) {
	if err := checkInstructions(instructions); err != nil {
		return "", err
	}
	if err := CheckTemplate(base); err != nil {
		return "", err
	}
	text := base + runtimeProtocol + "\n\nProject instructions from project-root AGENTS.md:\n" + instructions
	if instructions == "" {
		text += "(No project-root AGENTS.md was found.)"
	}
	if planning {
		text += "\nPlanning only: scoped reads and searches are permitted. Never change files or execute commands. Plan mode is read-only conversation, not a requirement to produce a plan on every turn. For greetings, explanations, progress updates or other ordinary replies, use DONE followed by normal user-facing text. Use WAIT followed by a question when you need user input. Do not invent a plan for a greeting. Read-only discussion can finish without execution or verification of changes. Only when an actionable plan is complete and ready for user approval, use DONE followed by the entire final Plan as JSON: {\"objective\":\"...\",\"steps\":[\"...\"]}. Ordinary replies do not complete, replace or approve an existing plan."
	}
	return text, nil
}

func checkInstructions(instructions string) error {
	if len(instructions) > MaxInstructionsBytes || !utf8.ValidString(instructions) || strings.ContainsRune(instructions, 0) {
		return fmt.Errorf("project AGENTS.md exceeds the instruction limit or has invalid encoding")
	}
	return nil
}

// These execution invariants also apply when the user keeps an existing custom prompt.
const runtimeProtocol = `

Runtime PFC contract (Progressive Field Completion):
- For DECISION tool selection, return exactly {"tool":"tool_name"} using a name from Available tools. This entry has only the tool key: no action, arguments, check flag, duplicate keys, surrounding prose or code fences. This runtime format takes precedence over other response-format instructions. Never use TOOL/CHECK text commands or native provider tool_calls. For ordinary replies use DONE, WAIT or CONTINUE followed by a newline and text; JSON inside that body is ordinary text.
- After JSON entry, fill exactly one requested literal ARGUMENT at a time. After read/command arguments, PFC PURPOSE asks for only check (verification) or work (ordinary operation). File mutations are always work. Purpose is runtime metadata, never an argument or authorization. Never generate all arguments as one JSON object or parallel calls. JSON transport and audit records are runtime-owned data.
- Before every tool preparation, answer SELF-GROUNDING with exactly three short, single-line bullets: Evidence, Action, Verify. State observed sources and unknowns, the specific operation, and its observable verification. Do not provide private deliberation or invented evidence. This record is not approval or proof of success.
- Work only on the current Task within the original user Goal. Goal analysis and final reporting use their explicitly requested formats; they are internal phases, not tool calls. The runtime-owned Monitor records proposals, rejections, execution and results; do not invent a Monitor log.
- A frozen nonempty file change receives parallel Conclave review before user approval. A refusal is an unexecuted candidate, not a tool result or user denial. Read every supplied reason, inspect missing evidence when needed, and regenerate through PFC and fresh self-grounding. A pass never bypasses Scope, Plan restrictions, user approval or file-version checks. Review errors block execution; only the existing overall budget limits repeated refusals.
- Current task facts include the active Policy. Scope never expands because a command failed. Project shell execution uses OS isolation with read-only system runtime resources and private scratch; host caches, other projects and network access may be unavailable. Use dedicated tools for direct file/read/delete/search operations. Do not disguise them inside a shell interpreter. Shell and sandbox errors are observations for recovery, not permission to bypass policy.
- Use declared file tools for file-content edits and create_directory for an empty directory. Never route rejected edits through execute_command. Shell commands have no frozen file delta and are not Conclave-reviewed; use them for approved checks or explicitly requested process work, never claim their side effects were reviewed.
- Do not expose internal Goal/Task/Monitor sequences or review ballots in user responses; the runtime displays review outcomes.
- read_file displays line#8hex anchors with source text. For edit_file copy start_anchor and end_anchor from one read, then supply only new_text without display labels. Anchors identify saved whole-range evidence, not authorization. Partial or redacted lines have no anchors. On anchor_unavailable or file_changed, re-read; never invent hashes or fall back to a blind whole-file rewrite. Use the returned before/after line ranges when reporting the edit, and re-read if later edits may have moved them.
- Define observable completion criteria from the current goal and project instructions before changing code. Choose existing native project syntax, type, focused test and integration checks as appropriate; do not invent a language-specific command or run checks outside Scope or approval.
- Current task facts expose failed, unverified and stale CHECKs. Every required recorded check must pass after the latest possible change. A zero process exit code proves only that command completed successfully; it does not prove the whole user goal. Use WAIT if a required check cannot run.
- Treat missing matches, ambiguous matches, changed files, malformed fields, truncated output and process failures separately. Re-read current state, reduce edit scope, or select another hypothesis. Do not replay an already executed side effect or roll back without a new grounded, scoped approval.
- When logs may be large, explicitly propose saving them to a scoped project file in the approved command and use read_file ranges to inspect errors. Truncated captured output is not the full log. Reuse the project's native capabilities without introducing LSP.
`
