// Purpose: Select a smaller generation output only after a provider reports a confirmed budget rejection.
// Pattern: Per-request recovery decision using exact counts or a lower bound.
// Usage: generate attempts one correction before forwarding capacity errors to the agent.
// Related: internal/provider/httpclient/request.go, internal/agent/run/loop.go.
package httpclient

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

func rejectedBudget(err error, sent, window int, canReduce bool) (int, error) {
	evidence, ok := transport.RejectedOutput(err, sent)
	if !ok {
		return 0, err
	}
	if evidence.Capacity > 0 {
		window = min(window, evidence.Capacity)
	}
	capacity := &modelapi.ContextLimitError{Cause: err, Capacity: evidence.Capacity}
	if !evidence.LowerBound {
		capacity.Input = evidence.Input
	}
	room := window - evidence.Input
	if !canReduce || room < 1 || sent < 2 {
		return 0, capacity
	}
	if evidence.LowerBound || evidence.Capacity == 0 && room >= sent {
		// A truncated input count gives only an upper bound on remaining space.
		// Leave headroom; a second rejection is handled by context compaction.
		return min(8192, max(1, room/2), max(1, sent/2)), nil
	}
	return min(room, sent-1), nil
}
