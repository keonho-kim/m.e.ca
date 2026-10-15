// Purpose: Define and validate finite Ralph-loop execution budgets.
// Pattern: Simple Module.
// Usage: TOML settings and Runner use Limits.
// Related: internal/agent/run/runner.go, internal/config/settings.go.
package run

import (
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

type Limits struct {
	MaxIterations    int `toml:"max_iterations"`
	MaxModelCalls    int `toml:"max_model_calls"`
	MaxSeconds       int `toml:"max_seconds"`
	MaxModelSeconds  int `toml:"max_model_seconds"`
	MaxContextBytes  int `toml:"max_context_bytes"`
	MaxArgumentBytes int `toml:"max_argument_bytes"`
	StallIterations  int `toml:"stall_iterations"`
}

func DefaultLimits() Limits {
	return Limits{MaxIterations: 128, MaxModelCalls: 1024, MaxSeconds: 14400, MaxModelSeconds: 600, MaxContextBytes: 256 * 1024 * 1024, MaxArgumentBytes: toolapi.MaxArgumentsBytes, StallIterations: 3}
}
func (l Limits) Check() error {
	if l.MaxIterations < 1 || l.MaxIterations > 4096 || l.MaxModelCalls < 1 || l.MaxModelCalls > 16384 || l.MaxSeconds < 1 || l.MaxSeconds > 86400 || l.MaxModelSeconds < 1 || l.MaxModelSeconds > 3600 || l.MaxContextBytes < 8192 || l.MaxContextBytes > 256*1024*1024 || l.MaxArgumentBytes < 1024 || l.MaxArgumentBytes > toolapi.MaxArgumentsBytes || l.StallIterations < 1 || l.StallIterations > 10 {
		return fmt.Errorf("invalid Ralph loop limits")
	}
	return nil
}
