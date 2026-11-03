// Purpose: Build native function schemas shared by request dialects.
// Pattern: Pure domain-to-wire conversion.
// Usage: Chat, Ollama and Responses request encoders share these declarations.
// Related: internal/provider/protocol/chat/request.go, internal/provider/protocol/responses/request.go.
package native

import toolapi "github.com/keonho-kim/meca/internal/agent/tool"

type Function struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

func Tools(definitions []toolapi.Definition) []Tool {
	var tools []Tool
	for _, d := range definitions {
		tools = append(tools, Tool{Type: "function", Function: Function{Name: d.Name, Description: d.Description, Parameters: d.JSONSchema()}})
	}
	return tools
}
