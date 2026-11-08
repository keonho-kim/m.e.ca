// Purpose: Define immutable skill instructions and reference evidence without filesystem dependencies.
// Pattern: Feature-owned value contracts.
// Usage: The skill reader, tool outcomes and agent runtime exchange these values.
// Related: internal/skills/reader.go, internal/agent/tool/execution.go, internal/agent/run/skills.go.
package instruction

type Metadata struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	ManualOnly  bool     `json:"manual_only,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

type Document struct {
	Metadata
	Revision string `json:"revision"`
	Body     string `json:"body"`
}

type Selector struct{ ID, Path string }

// ReferenceRead identifies text already returned by one native read result.
type ReferenceRead struct {
	ID, Path, Revision string
	Offset             int64
}
