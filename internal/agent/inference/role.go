// Purpose: Name generation responsibilities independently of providers.
// Pattern: Simple Module.
// Usage: Runtime phases and diagnostics select a Role before calling the model.
// Related: internal/agent/inference/settings.go, internal/agent/tool/schema.go.
package inference

type Role string

const (
	Agent      Role = "agent"
	Compaction Role = "compaction"
	Diagnostic Role = "diagnostic"
)
