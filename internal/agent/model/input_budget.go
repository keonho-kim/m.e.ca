// Purpose: Describe the token budget and input measurement used for a model request or compaction decision.
// Pattern: Model boundary values.
// Usage: Provider preparation and compaction report counts and their provenance to generation accounting.
// Related: internal/agent/compact/controller.go, internal/agent/run/attempt_metrics.go, internal/agent/record/events.go.
package model

// InputBudget describes the context and output reservation used by the adapter.
// InputTokens is zero when the provider did not tokenize the request.
type InputBudget struct {
	Window, InputTokens, OutputTokens int
}

type ContextMeasurement struct {
	Input        int64  `json:"input_tokens"`
	Source       string `json:"input_source"`
	Window       int    `json:"context_window,omitempty"`
	OutputBudget int    `json:"output_budget,omitempty"`
}

func (m *ContextMeasurement) Valid() bool {
	if m == nil {
		return true
	}
	if m.Input < 0 || m.Window < 0 || m.OutputBudget < 0 {
		return false
	}
	switch m.Source {
	case "usage", "tokenizer", "estimate", "provider", "watermark", "watermark_estimate":
		return true
	}
	return false
}
