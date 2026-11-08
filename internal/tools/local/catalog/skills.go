// Purpose: Declare metadata discovery, instruction reads and confined skill reference tools.
// Pattern: Declarative native tool contracts.
// Usage: The fixed local catalog includes these definitions in every execution mode.
// Related: internal/tools/local/skills.go, internal/skills/reader.go.
package catalog

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/skills"
)

func skillSpecifications() []Specification {
	id := toolapi.Parameter{Name: "id", Description: "Skill ID returned by list_skills or load_skill, including opaque IDs from path loads; or an unqualified name using project, MECA and shared priority.", Kind: toolapi.Text, MaxBytes: 512}
	return []Specification{
		{"list_skills", "Listing skills", "List installed skill names, descriptions, origins and manual-only flags before selecting instructions for substantive work. Metadata only; use load_skill to read instructions. Reuse relevant results within this request. Invalid entries have diagnostics. Continue with next_cursor to see more.", []toolapi.Parameter{
			{Name: "cursor", Description: "Empty for the first page; otherwise next_cursor from the previous page. Pages share one metadata snapshot; restart an expired listing with an empty cursor.", Kind: toolapi.Text, AllowEmpty: true, MaxBytes: 64},
			{Name: "limit", Description: "Page item limit, default 32, maximum 32.", Kind: toolapi.Integer, Max: 32},
		}, policy.Read},
		{"load_skill", "Reading skill", "Read SKILL.md instructions by id OR path, never both. Returns id, body, metadata and a document revision. Use the returned id for read_skill_file, especially after a path load. Read relevant instructions before planning dependent actions. Explicit user #name mentions are loaded automatically; manual-only skills require such a mention. Relative paths start at project_root; ~/ uses the server or host account home. This only reads instructions and never executes scripts.", []toolapi.Parameter{id,
			{Name: "path", Description: "Skill directory or SKILL.md path. Alternative to id.", Kind: toolapi.LiteralPath, MaxBytes: 4096},
		}, policy.Read},
		{"read_skill_file", "Reading skill reference", "Read a reference, script or asset inside a verified skill directory. Paths are relative to that skill root, not project_root. Manual-only skills require an explicit user #mention for reference reads too. Start at offset 0 without revision. While next_offset is positive, keep the same id and path and pass offset=next_offset with the revision returned by this tool. A zero or absent next_offset means the file is complete. If the file version changed, restart at offset 0 without the old revision. Image assets use the existing vision decoder when available; other binary files return metadata only. Reading scripts does not run them.", []toolapi.Parameter{
			{Name: id.Name, Description: id.Description, Kind: id.Kind, Required: true, MaxBytes: id.MaxBytes},
			{Name: "path", Description: "Relative reference path within the skill directory.", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096},
			{Name: "offset", Description: "Byte offset, default zero; use next_offset and revision to continue the same file version.", Kind: toolapi.Integer, Max: skills.MaxResourceFileBytes},
			{Name: "revision", Description: "Copy the revision returned by read_skill_file's first page, not load_skill's document revision. Required when offset is nonzero; omit when restarting at zero.", Kind: toolapi.Text, MaxBytes: 64},
			{Name: "limit", Description: "Maximum text bytes, default and maximum 65536.", Kind: toolapi.Integer, Max: skills.MaxResourceBytes},
		}, policy.Read},
	}
}
