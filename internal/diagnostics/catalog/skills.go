// Purpose: Own skill read, pagination and retained-instruction diagnostics.
// Pattern: Declarative diagnostic catalog.
// Usage: Skill readers and request owners format these messages through diagnostics.
// Related: internal/skills, internal/agent/run/skills.go.
package catalog

const (
	SkillsMetadataUnavailable      Message = "SKILL.md or invocation metadata is missing or inaccessible"
	SkillsExtensionIgnored         Message = "Execution extension %s is not interpreted; MECA reads Markdown instructions and keeps its own tool policy."
	SkillsResourceRevisionRequired Message = "%w: continuation requires the resource revision returned by its first page"
	SkillsResourceRevisionChanged  Message = "%w: skill reference changed; restart from offset zero"
	SkillsInstructionBudget        Message = "current-request skill instructions exceed their retained byte budget"
)
