// Purpose: Share one decision's remaining input budget among all requested document pages.
// Pattern: Execution-owned estimate using existing provider usage and output reservations.
// Usage: askResponse calculates the allowance once; drive passes it through pending tool calls.
// Related: internal/agent/run/generate.go, internal/tools/browser/webfetch/session.go.
package run

import (
	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

const maxWebFetchDecisionBytes = 64 << 10

func (s *execution) pageAllowance(request modelapi.Request, reply modelapi.Response, budget modelapi.InputBudget, usage modelapi.TokenUsage, wireBytes int64) int {
	count := 0
	for _, call := range reply.Calls {
		if call.Name == toolapi.WebFetch {
			count++
		}
	}
	if count == 0 {
		return -1
	}
	assistant := modelapi.Message{Content: reply.Text, Reasoning: reply.Reasoning, ReasoningState: reply.ReasoningState, Calls: reply.Calls}
	assistantBytes := int64(assistant.ContextBytes())
	// Include the added message and continuation framing outside each payload.
	framing := int64(256 * (count + 1))
	remaining := int64(maxWebFetchDecisionBytes)
	if s.limits.MaxContextBytes > 0 {
		used := assistantBytes + framing
		for _, message := range request.Messages {
			used += int64(message.ContextBytes())
		}
		remaining = min(remaining, int64(s.limits.MaxContextBytes)-used)
	}
	window := budget.Window
	if memory := s.runner.Memory; memory != nil && memory.ContextWindow > 0 && (window == 0 || memory.ContextWindow < window) {
		window = memory.ContextWindow
	}
	if window == 0 {
		return int(max(0, remaining) / int64(count))
	}
	output := budget.OutputTokens
	if output == 0 {
		var err error
		output, err = modelapi.OutputBudget(request.MaxOutput, window, 0, 0)
		if err != nil {
			return 0
		}
	}
	input := max(usage.Input, int64(budget.InputTokens))
	if input <= 0 {
		if wireBytes <= 0 {
			wireBytes = modelapi.TextBytes(request)
		}
		input = compact.EstimateBytes(wireBytes).Amount
	}
	used := input + max(usage.Output, compact.EstimateBytes(assistantBytes).Amount) + framing
	// Charge each new serialized byte against one token of headroom. Provider
	// usage may be unavailable, so the existing capacity recovery still applies.
	remaining = min(remaining, int64(window)-int64(output)-used)
	return int(max(0, remaining) / int64(count))
}
