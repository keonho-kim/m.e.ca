// Purpose: Name generation responsibilities and classify literal PFC fields.
// Pattern: Simple Module.
// Usage: Runtime phases and diagnostics select a Role before calling the model.
// Related: internal/agent/inference/settings.go, internal/agent/tool/schema.go.
package inference

import toolapi "github.com/keonho-kim/v-code/internal/agent/tool"

type Role string

const (
	Agent      Role = "agent"
	Plan       Role = "plan"
	Goal       Role = "goal"
	PFCValue   Role = "pfc_value"
	PFCContent Role = "pfc_content"
	Grounding  Role = "grounding"
	Conclave   Role = "conclave"
	Compaction Role = "compaction"
	Report     Role = "report"
	Diagnostic Role = "diagnostic"
)

func FieldRole(parameter toolapi.Parameter) Role {
	if parameter.Kind == toolapi.Text {
		switch parameter.Name {
		case "content", "new_text", "command":
			return PFCContent
		}
	}
	return PFCValue
}
