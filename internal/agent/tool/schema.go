// Purpose: Declare staged tool parameter types and fixed tool definitions.
// Pattern: Tool schema.
// Usage: Dialogue and prompt code use the same parameters as the local tool catalog.
// Related: internal/agent/pfc/pfc.go, internal/tools/local/catalog.go.

package tool

import "github.com/keonho-kim/v-code/internal/policy"

// ParameterKind is the literal value type requested in one model reply.
type ParameterKind string

const (
	Text        ParameterKind = "text"
	LiteralPath ParameterKind = "literal_path"
	Integer     ParameterKind = "integer"
	LineAnchor  ParameterKind = "line_anchor"
)

type Parameter struct {
	Name, Question       string
	Kind                 ParameterKind
	Required, AllowEmpty bool
	Min, Max, MaxBytes   int
}
type Definition struct {
	Name, Description string
	Effect            policy.Effect
	Parameters        []Parameter
}
