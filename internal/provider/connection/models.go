// Purpose: Describe discovered models and selected model settings with identity evidence.
// Pattern: Model-selection values.
// Usage: HTTP discovery, setup and SQLite persist these provider-neutral values.
// Related: internal/provider/setup/setup.go, internal/provider/identity/report.go, internal/storage/sqlite/selection.go.

package connection

import "github.com/keonho-kim/meca/internal/provider/identity"

type Capabilities struct {
	Reasoning bool `json:"reasoning"`
	Thinking  bool `json:"thinking"`
	Chat      bool `json:"chat"`
	Streaming bool `json:"streaming"`
}

type ModelInfo struct {
	Notice        string     `json:"notice,omitempty"`
	Reasoning     *Reasoning `json:"reasoning,omitempty"`
	ID            string     `json:"id"`
	ContextWindow int        `json:"context_window"`
	MaxInput      int        `json:"max_input"`
	MaxOutput     int        `json:"max_output"`
}

type Selection struct {
	Details           *ModelInfo       `json:"details,omitempty"`
	Identity          *identity.Report `json:"identity,omitempty"`
	ReasoningEffort   *string          `json:"reasoning_effort,omitempty"`
	Connection        Connection       `json:"connection"`
	Model             string           `json:"model"`
	Thinking          *bool            `json:"thinking,omitempty"`
	ThinkingSupported bool             `json:"thinking_supported"`
}

// Effort returns the effective request value; empty delegates to provider defaults.
func (s Selection) Effort() string {
	if s.ReasoningEffort == nil {
		return ""
	}
	return *s.ReasoningEffort
}
