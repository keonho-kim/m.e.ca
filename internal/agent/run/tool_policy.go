// Purpose: Apply execution-local read-only restrictions before the ordinary tool policy.
// Pattern: Simple policy composition.
// Usage: Tool advertisement and execution share this eligibility decision.
// Related: internal/agent/run/context.go, internal/agent/run/action.go, internal/policy/policy.go.
package run

import (
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *execution) allowTool(effect policy.Effect) error {
	// Command eligibility advertises the Plan query entry; action and local
	// preparation parse its exact arguments before any execution.
	if s.planning && effect == policy.Command {
		return s.runner.Settings.Check()
	}
	_, err := policy.Decide(s.runner.Settings, s.planning, effect)
	return err
}
