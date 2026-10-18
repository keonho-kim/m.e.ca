// Purpose: Declare native tool parameter types and fixed tool definitions.
// Pattern: Tool schema.
// Usage: Provider adapters use the same parameters as the local tool catalog.
// Related: internal/agent/tool/arguments.go, internal/tools/local/catalog.go.

package tool

import "github.com/keonho-kim/v-code/internal/policy"

// ParameterKind is the accepted type of one field in a complete argument object.
type ParameterKind string

const (
	Text        ParameterKind = "text"
	TextList    ParameterKind = "text_list"
	LiteralPath ParameterKind = "literal_path"
	Integer     ParameterKind = "integer"
	ObjectList  ParameterKind = "object_list"
)

type Parameter struct {
	Fields               []Parameter
	Values               []string
	Name, Description    string
	Kind                 ParameterKind
	Required, AllowEmpty bool
	Min, Max, MaxBytes   int
	MaxChars             int
}
type Definition struct {
	Name, Description string
	Effect            policy.Effect
	Parameters        []Parameter
}
