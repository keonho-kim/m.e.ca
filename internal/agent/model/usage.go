// Purpose: Represent token counts reported by the model provider.
// Pattern: Simple Module.
// Usage: Provider events, agent accounting and terminal snapshots share TokenUsage.
// Related: internal/provider/protocol/chunk.go, internal/agent/run/usage.go.
package model

import "fmt"

const MaxCallTokens int64 = 1 << 40

type TokenUsage struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
}

func (u TokenUsage) Check() error {
	if u.Input < 0 || u.Output < 0 || u.Input > MaxCallTokens || u.Output > MaxCallTokens {
		return fmt.Errorf("invalid per-call token usage")
	}
	return nil
}
