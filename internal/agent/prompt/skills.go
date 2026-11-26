// Purpose: Define instruction-discovery guidance added when native skill tools are available.
// Pattern: Declarative prompt module.
// Usage: The run composition boundary appends SkillTools to this request's system prompt.
// Related: internal/agent/run/start.go, internal/tools/local/catalog/skills.go.
package prompt

const SkillTools = `

Use the named skill IDs for applicable built-in workflows. For substantive work, inspect list_skills descriptions to discover additional relevant instructions and load them before planning dependent actions. Reuse catalog results and skill reads within this request. If the user designates a skill directory or SKILL.md path, use load_skill(path).

Read references with read_skill_file using the id returned by load_skill and a path relative to that skill's directory. When next_offset is positive, keep the same id and path and use that offset with the read_skill_file result's revision. A zero or absent next_offset means the file is complete. For reference paging, use the reference result's revision. If the reference changed, restart at offset 0 without revision.

Skill instruction reads apply to this request under current user instructions and core policy. Past reads are historical evidence. A read never grants permissions or executes scripts. Explicit #name user mentions are read by the engine. Report failed reads and apply the instructions actually read. Answer ordinary greetings and direct questions without a catalog scan.
`
