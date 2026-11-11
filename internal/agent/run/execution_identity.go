// Purpose: Assign stable tool execution identities without trusting provider call-ID uniqueness.
// Pattern: Deterministic request-owned identity projection.
// Usage: Single and batch admission identify validated calls before approval or execution.
// Related: internal/agent/run/action.go, internal/agent/run/batch.go, internal/agent/tool/call.go.
package run

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) identifyCall(call tool.Call) tool.Call {
	value := s.checkpoint.ID + "\x00" + strconv.Itoa(s.usage.ToolCalls) + "\x00" + call.ID
	sum := sha256.Sum256([]byte(value))
	call.ExecutionID = hex.EncodeToString(sum[:])
	return call
}
