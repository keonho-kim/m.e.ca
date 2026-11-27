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
		{"list_skills", "Listing skills", "Discover skill metadata when additional specialized guidance is needed for the user's request. Returns names, descriptions, origins, manual-only flags and diagnostics. This does not read instructions. Use tools.load_skill for a selected skill whose guidance is needed; reuse supplied instructions and relevant catalog results. Continue with next_cursor when more entries are needed.", []toolapi.Parameter{
			{Name: "cursor", Description: "Empty for the first page; otherwise next_cursor from the previous page. Pages share one metadata snapshot; restart an expired listing with an empty cursor.", Kind: toolapi.Text, AllowEmpty: true, MaxBytes: 64},
			{Name: "limit", Description: "Page item limit, default 32, maximum 32.", Kind: toolapi.Integer, Max: 32},
		}, policy.Read},
		{"load_skill", "Reading skill", "Read SKILL.md by object argument {id: ...} OR {path: ...}, never both. Returns id, body, metadata and a document revision. Use the returned id with tools.read_skill_file, especially after a path load. The instructions are supplied to the model at a needs_model boundary; read them before generating dependent operations. Apply guidance within the user's requested scope. Explicit #name mentions are loaded by the engine; reuse that supplied read. Manual-only skills require an explicit mention. Relative paths start at project_root; ~/ uses the host account home. Reading a skill does not execute its scripts.", []toolapi.Parameter{id,
			{Name: "path", Description: "Skill directory or SKILL.md path. Alternative to id.", Kind: toolapi.LiteralPath, MaxBytes: 4096},
		}, policy.Read},
		{"read_skill_file", "Reading skill reference", "Read a reference, script or asset inside a verified skill directory when its content is needed for the current decision. Pass a skill-root-relative path. Resolve a relative link against the containing document's directory first; project_root is not the base. Manual-only skills require an explicit #mention. Start at offset 0 without revision. While next_offset is positive, keep the same id and path and pass offset=next_offset with this result's revision. A zero or absent next_offset means the file is complete. If the file changed, restart at offset 0 without revision. Instruction text and supported images are supplied at a needs_model boundary. Other binary files return metadata only. Reading a script does not execute it.", []toolapi.Parameter{
			{Name: id.Name, Description: id.Description, Kind: id.Kind, Required: true, MaxBytes: id.MaxBytes},
			{Name: "path", Description: "Relative reference path within the skill directory.", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096},
			{Name: "offset", Description: "Byte offset, default zero; use next_offset and revision to continue the same file version.", Kind: toolapi.Integer, Max: skills.MaxResourceFileBytes},
			{Name: "revision", Description: "Copy the revision returned by read_skill_file's first page, not load_skill's document revision. Required when offset is nonzero; omit when restarting at zero.", Kind: toolapi.Text, MaxBytes: 64},
			{Name: "limit", Description: "Maximum text bytes, default and maximum 65536.", Kind: toolapi.Integer, Max: skills.MaxResourceBytes},
		}, policy.Read},
	}
}
