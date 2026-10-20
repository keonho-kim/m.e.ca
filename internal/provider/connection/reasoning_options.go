// Purpose: Resolve server-advertised choices or serving API request options without model-name rules.
// Pattern: Provider protocol values.
// Usage: Setup, selection validation and request encoding share ReasoningOptions.
// Related: internal/provider/connection/reasoning.go, internal/provider/protocol/controls.go.
package connection

// ReasoningOptions describes accepted request syntax, not proof that a model obeys it.
// Explicit paired generation diagnostics verify observable ON/OFF behavior separately.
func ReasoningOptions(kind ProviderKind, info *ModelInfo) Reasoning {
	if info != nil && info.Reasoning != nil {
		return *info.Reasoning
	}
	switch kind {
	case VLLM, SGLang:
		return Reasoning{On: true, Off: true, Levels: []string{"minimal", "low", "medium", "high", "xhigh", "max"}}
	default:
		return Reasoning{}
	}
}
