// Purpose: Apply execution-local read-only restrictions before the ordinary tool policy.
// Pattern: Simple policy composition.
// Usage: Tool advertisement and execution share this eligibility decision.
// Related: internal/agent/run/context.go, internal/agent/run/action.go, internal/policy/policy.go.
package run

import (
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *execution) allowTool(effect policy.Effect) error {
	_, err := policy.Decide(s.runner.Settings, s.planning, effect)
	return err
}
