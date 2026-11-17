// Purpose: Assign stable tool execution identities without trusting provider call-ID uniqueness.
// Pattern: Request-owned candidate identity allocation.
// Usage: Single, batch and rejected candidates receive identities before publication.
// Related: internal/agent/run/action.go, internal/agent/run/batch.go, internal/agent/tool/call.go.
package run

import "github.com/keonho-kim/meca/internal/agent/tool"

func (s *execution) identifyCall(call tool.Call) tool.Call {
	call.ExecutionID = s.order.Next()
	// Only the runtime scheduler may assign a parallel cohort.
	call.BatchID = ""
	return call
}
